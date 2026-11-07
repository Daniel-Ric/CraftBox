package resourcepacks

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime/pprof"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"
)

const packChunkSize = 1024 * 128

const packRequestWindow = 100

type downloadingPack struct {
	chunkSize  uint32
	size       uint64
	newFrag    chan *packet.ResourcePackChunkData
	done       chan struct{}
	contentKey string

	ID      uuid.UUID
	Version string
}

type uploadingPack struct {
	Pack          *resource.Pack
	currentOffset uint64
}

type ResourcePackHandler struct {
	ctx            context.Context
	log            *logrus.Entry
	cancelDownload context.CancelCauseFunc
	downloadError  sync.Once
	httpClient     *http.Client

	OnError func(error)

	Server minecraft.IConn
	Client minecraft.IConn

	OnResourcePacksInfoCB func()

	OnFinishedPack func(*resource.Pack) error

	cache      PackCache
	addedPacks []*resource.Pack

	resourcePacks     []*resource.Pack
	finishedPacks     []string
	lockResourcePacks sync.Mutex

	packsDownloading atomic.Int32
	downloadingPacks map[uuid.UUID]downloadingPack
	awaitingPacks    map[uuid.UUID]*downloadingPack
	downloadMu       sync.Mutex
	stallSnapshot    sync.Once

	receivedRemotePackInfo chan struct{}
	remotePacksInfo        *packet.ResourcePacksInfo

	receivedRemoteStack chan struct{}
	remoteStack         *packet.ResourcePackStack

	knowPacksRequestedFromServer chan struct{}
	packsRequestedFromServer     []string

	clientHasRequested bool
	allPacksDownloaded bool

	clientQueueMu      sync.Mutex
	pendingClientPacks map[string]struct{}
	nextPackToClient   chan *resource.Pack
	uploads            map[uuid.UUID]*uploadingPack
	uploadLock         sync.Mutex
	clientDone         chan struct{}
}

func NewResourcePackHandler(ctx context.Context, addedPacks []*resource.Pack) *ResourcePackHandler {
	ctx, cancel := context.WithCancelCause(ctx)
	r := &ResourcePackHandler{
		ctx:            ctx,
		cancelDownload: cancel,
		log:            logrus.WithField("part", "ResourcePacks"),
		addedPacks:     addedPacks,

		cache:                  &packCache{},
		receivedRemotePackInfo: make(chan struct{}),
		receivedRemoteStack:    make(chan struct{}),
		downloadingPacks:       make(map[uuid.UUID]downloadingPack),
		awaitingPacks:          make(map[uuid.UUID]*downloadingPack),
	}
	return r
}

func (r *ResourcePackHandler) failDownload(err error) {

	if r.ctx.Err() != nil || (r.Server != nil && r.Server.Context().Err() != nil) {
		return
	}
	r.downloadError.Do(func() {
		r.log.Error(err)
		if r.OnError != nil {
			r.OnError(err)
		}
		r.cancelDownload(err)
		if r.OnError == nil && r.Server != nil {
			_ = r.Server.Close()
		}
	})
}

func (r *ResourcePackHandler) SetCache(c PackCache) {
	r.cache = c
}

// SetHTTPClient is configured before any pack callbacks start. Keep the
// existing HTTP/1.1 pack-download compatibility while retaining pinned dialing.
func (r *ResourcePackHandler) SetHTTPClient(client *http.Client) {
	copyClient := *client
	if transport, ok := client.Transport.(*http.Transport); ok {
		copyTransport := transport.Clone()
		copyTransport.ForceAttemptHTTP2 = false
		if copyTransport.TLSClientConfig == nil {
			copyTransport.TLSClientConfig = &tls.Config{}
		} else {
			copyTransport.TLSClientConfig = copyTransport.TLSClientConfig.Clone()
		}
		copyTransport.TLSClientConfig.NextProtos = []string{"http/1.1"}
		copyClient.Transport = copyTransport
	}
	r.httpClient = &copyClient
	context.AfterFunc(r.ctx, copyClient.CloseIdleConnections)
}

func (r *ResourcePackHandler) SetServer(c minecraft.IConn) {
	r.Server = c
}

func (r *ResourcePackHandler) SetClient(c minecraft.IConn) {
	r.Client = c
	r.knowPacksRequestedFromServer = make(chan struct{})
	r.clientDone = make(chan struct{})
}

var httpClient = http.Client{
	Transport: &http.Transport{
		ForceAttemptHTTP2: false,
		TLSClientConfig: &tls.Config{
			NextProtos: []string{"http/1.1"},
		},
	},
}

func (r *ResourcePackHandler) downloadFromUrl(pack protocol.TexturePackInfo) error {
	r.log.Infof("Downloading Resourcepack: %s", pack.DownloadURL)

	f, err := r.cache.Create(pack.UUID, pack.Version)
	if err != nil {
		return err
	}
	defer f.Close()

	req, err := http.NewRequestWithContext(r.ctx, "GET", pack.DownloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "libhttpclient/1.0.0.0")
	client := r.httpClient
	if client == nil {
		client = &httpClient
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("resource pack CDN returned HTTP %d", res.StatusCode)
	}

	size, err := io.Copy(f, res.Body)
	if err != nil {
		return err
	}

	newPack, err := resource.FromReaderAt(f, size)
	if err != nil {
		return err
	}
	if len(pack.ContentKey) > 0 {
		newPack = newPack.WithContentKey(pack.ContentKey)
	}
	newPack, err = utils.PackFromBase(newPack)
	if err != nil {
		return err
	}

	if err := f.Move(); err != nil {
		return err
	}
	return r.finishedPack(pack.UUID.String()+"_"+pack.Version, newPack)
}

func (r *ResourcePackHandler) finishedPack(idVer string, pack *resource.Pack) error {
	messages.SendEvent(&messages.EventResourcePackAvailable{Pack: pack})
	r.lockResourcePacks.Lock()
	r.resourcePacks = append(r.resourcePacks, pack)
	r.finishedPacks = append(r.finishedPacks, idVer)
	err := r.OnFinishedPack(pack)
	r.lockResourcePacks.Unlock()
	if err != nil {
		return err
	}
	if r.knowPacksRequestedFromServer != nil {

		go r.deliverFinishedPack(idVer, pack)
	}
	if r.packsDownloading.Add(-1) == 0 {
		r.Server.Expect(packet.IDResourcePackStack)
		_ = r.Server.WritePacket(&packet.ResourcePackClientResponse{Response: packet.PackResponseAllPacksDownloaded})
	}

	return nil
}

func (r *ResourcePackHandler) deliverFinishedPack(idVer string, pack *resource.Pack) {
	select {
	case <-r.knowPacksRequestedFromServer:
	case <-r.ctx.Done():
		return
	}
	r.clientQueueMu.Lock()
	defer r.clientQueueMu.Unlock()
	if _, requested := r.pendingClientPacks[idVer]; !requested {
		return
	}
	select {
	case r.nextPackToClient <- pack:
	case <-r.ctx.Done():
		return
	}
	delete(r.pendingClientPacks, idVer)
	if len(r.pendingClientPacks) == 0 {
		close(r.nextPackToClient)
	}
}

func (r *ResourcePackHandler) OnResourcePacksInfo(pk *packet.ResourcePacksInfo) error {
	if r.OnResourcePacksInfoCB != nil {
		r.OnResourcePacksInfoCB()
	}

	packsToDownload := make([]string, 0, len(pk.TexturePacks))

	var urlDownloads []protocol.TexturePackInfo
	for _, pack := range pk.TexturePacks {
		packID := pack.UUID.String() + "_" + pack.Version
		r.log.Debugf("remote pack %s: %d bytes, cached=%t, URL=%t", packID, pack.Size, r.cache.Has(pack.UUID, pack.Version), pack.DownloadURL != "")

		_, alreadyDownloading := r.downloadingPacks[pack.UUID]
		if alreadyDownloading {
			r.log.Warnf("duplicate texture pack entry %v in resource pack info", pack.UUID)
			continue
		}

		if r.cache.Has(pack.UUID, pack.Version) {
			newPack, err := r.cache.Get(pack.UUID, pack.Version)
			if err != nil {
				return fmt.Errorf("opening cached Resourcepack: %s (delete packcache as a fix)", err)
			}

			if len(pack.ContentKey) > 0 {
				newPack = newPack.WithContentKey(pack.ContentKey)
			}
			newPack, err = utils.PackFromBase(newPack)
			if err != nil {
				return err
			}

			r.lockResourcePacks.Lock()
			r.resourcePacks = append(r.resourcePacks, newPack)
			r.finishedPacks = append(r.finishedPacks, pack.UUID.String()+"_"+pack.Version)
			messages.SendEvent(&messages.EventResourcePackAvailable{Pack: newPack})
			err = r.OnFinishedPack(newPack)
			r.lockResourcePacks.Unlock()
			if err != nil {
				return err
			}
			continue
		}

		if pack.DownloadURL != "" {
			urlDownloads = append(urlDownloads, pack)
			r.packsDownloading.Add(1)
			continue
		}

		packsToDownload = append(packsToDownload, packID)
		r.downloadingPacks[pack.UUID] = downloadingPack{
			size:       pack.Size,
			newFrag:    make(chan *packet.ResourcePackChunkData),
			done:       make(chan struct{}),
			contentKey: pack.ContentKey,
			ID:         pack.UUID,
			Version:    pack.Version,
		}
		r.packsDownloading.Add(1)
	}

	if len(urlDownloads) > 0 {
		go func() {
			for _, dl := range urlDownloads {
				if err := r.downloadFromUrl(dl); err != nil {
					r.failDownload(fmt.Errorf("download resource pack %s: %w", dl.UUID, err))
					return
				}
			}
		}()
	}

	r.remotePacksInfo = pk
	close(r.receivedRemotePackInfo)
	r.log.Debugf("received remote pack infos: %d packs, %d packet downloads, %d URL downloads", len(pk.TexturePacks), len(packsToDownload), len(urlDownloads))

	if r.knowPacksRequestedFromServer != nil {
		select {
		case <-r.knowPacksRequestedFromServer:
		case <-r.ctx.Done():
			return context.Cause(r.ctx)
		}
	}

	if len(packsToDownload) == 0 {
		r.Server.Expect(packet.IDResourcePackStack)
		if len(urlDownloads) == 0 {
			return r.Server.WritePacket(&packet.ResourcePackClientResponse{Response: packet.PackResponseAllPacksDownloaded})
		}

		return nil
	} else {
		r.log.Debugf("requesting %d resource packs from server", len(packsToDownload))
		r.Server.Expect(packet.IDResourcePackDataInfo, packet.IDResourcePackChunkData)
		err := r.Server.WritePacket(&packet.ResourcePackClientResponse{
			Response:        packet.PackResponseSendPacks,
			PacksToDownload: packsToDownload,
		})
		if err != nil {
			return err
		}

	}
	return nil
}

func (r *ResourcePackHandler) downloadResourcePack(pk *packet.ResourcePackDataInfo, pack *downloadingPack) error {
	chunkCount := int32(pk.Size / uint64(pk.DataChunkSize))
	if pk.Size%uint64(pk.DataChunkSize) != 0 {
		chunkCount++
	}

	f, err := r.cache.Create(pack.ID, pack.Version)
	if err != nil {
		return err
	}
	defer f.Close()
	r.log.Debugf("downloading pack %s: %d bytes, chunk size %d, chunks %d", pk.UUID, pk.Size, pk.DataChunkSize, pk.ChunkCount)

	requestChunk := func(index int32) error {
		if r.ctx.Err() != nil {
			return context.Cause(r.ctx)
		}
		if index == 0 {
			r.log.Debugf("requesting first chunk for pack %s", pk.UUID)
		}
		if err := r.Server.WritePacket(&packet.ResourcePackChunkRequest{

			UUID:       pk.UUID,
			ChunkIndex: index,
		}); err != nil {
			return err
		}
		if index == 0 {
			r.log.Debugf("first chunk request queued for pack %s", pk.UUID)
		}
		return nil
	}

	nextRequest := min(int32(packRequestWindow), chunkCount)
	for index := range nextRequest {
		if err := requestChunk(index); err != nil {
			return err
		}
	}
	r.log.Debugf("initial resource pack request window queued for %s: %d/%d chunks", pk.UUID, nextRequest, chunkCount)

	received := make(map[uint32]struct{})
	var dataWritten uint64
	var firstMissing uint32
	for count := range chunkCount {
		waitingIndex, waitingBytes, waitingCount := firstMissing, dataWritten, count
		stalled := time.AfterFunc(10*time.Second, func() {
			r.log.Warnf("resource pack %s: still waiting for chunk %d after 10s (received %d/%d chunks, %d/%d bytes)", pk.UUID, waitingIndex, waitingCount, chunkCount, waitingBytes, pack.size)
			r.writeStallSnapshot()
		})
		var frag *packet.ResourcePackChunkData
		select {
		case <-r.ctx.Done():
			stalled.Stop()
			return context.Cause(r.ctx)
		case <-r.Server.Context().Done():
			stalled.Stop()
			return net.ErrClosed
		case frag = <-pack.newFrag:
		}
		stalled.Stop()
		index := frag.ChunkIndex
		if _, exists := received[index]; exists {
			return fmt.Errorf("duplicate resource pack chunk index %d", index)
		}
		if index >= uint32(nextRequest) {
			return fmt.Errorf("unrequested resource pack chunk index %d (requested through %d)", index, nextRequest-1)
		}
		offset := uint64(index) * uint64(pack.chunkSize)
		length := min(uint64(pack.chunkSize), pack.size-offset)
		if frag.DataOffset != offset {
			return fmt.Errorf("resource pack chunk %d offset %d != expected %d", index, frag.DataOffset, offset)
		}
		if uint64(len(frag.Data)) != length {
			return fmt.Errorf("resource pack chunk %d length %d != expected %d", index, len(frag.Data), length)
		}
		if _, err := f.WriteAt(frag.Data, int64(offset)); err != nil {
			return err
		}
		received[index] = struct{}{}
		dataWritten += uint64(len(frag.Data))
		for firstMissing < uint32(chunkCount) {
			if _, ok := received[firstMissing]; !ok {
				break
			}
			firstMissing++
		}
		if count == 0 || (count+1)%100 == 0 || count+1 == chunkCount {
			r.log.Debugf("resource pack %s: received chunk %d, progress %d/%d chunks (%d/%d bytes), next missing %d", pk.UUID, index, count+1, chunkCount, dataWritten, pack.size, firstMissing)
		}
		if nextRequest < chunkCount {
			if err := requestChunk(nextRequest); err != nil {
				return err
			}
			nextRequest++
		}
	}
	if dataWritten != pack.size {
		return fmt.Errorf("incorrect resource pack size: expected %v, but got %v", pack.size, dataWritten)
	}
	h := sha256.New()
	if _, err := io.Copy(h, io.NewSectionReader(f, 0, int64(pack.size))); err != nil {
		return fmt.Errorf("read resource pack for checksum: %w", err)
	}

	sum := h.Sum(nil)
	if !bytes.Equal(pk.Hash, sum) {
		return fmt.Errorf("resource pack download error, hash mismatch in download %s", pk.UUID)
	}

	newPack, err := resource.FromReaderAt(f, int64(pack.size))
	if err != nil {
		return fmt.Errorf("invalid full resource pack data for UUID %v: %v", pk.UUID, err)
	}
	newPack, err = utils.PackFromBase(newPack.WithContentKey(pack.contentKey))
	if err != nil {
		return fmt.Errorf("invalid full resource pack data for UUID %v: %v", pk.UUID, err)
	}

	err = f.Move()
	if err != nil {
		return err
	}

	err = r.finishedPack(pack.ID.String()+"_"+pack.Version, newPack)
	if err != nil {
		return err
	}
	return nil
}

func (r *ResourcePackHandler) writeStallSnapshot() {
	r.stallSnapshot.Do(func() {
		path := utils.PathData("resourcepack-stall.txt")
		f, err := utils.CreateShared(path)
		if err != nil {
			r.log.Warnf("could not create resource pack stall diagnostics: %v", err)
			return
		}
		defer f.Close()
		if _, err = fmt.Fprintf(f, "CraftBox %s\nResource pack handshake stalled at %s\n\n", utils.Version, time.Now().Format(time.RFC3339)); err == nil {
			if diagnostics, ok := r.Server.(interface{ HandshakeDiagnostics() string }); ok {
				_, err = fmt.Fprintln(f, diagnostics.HandshakeDiagnostics())
			}
		}
		if err == nil {
			err = pprof.Lookup("goroutine").WriteTo(f, 1)
		}
		if err != nil {
			r.log.Warnf("could not write resource pack stall diagnostics: %v", err)
			return
		}
		r.log.Warnf("resource pack stall diagnostics saved to %s", path)
	})
}

func (r *ResourcePackHandler) OnResourcePackDataInfo(pk *packet.ResourcePackDataInfo) error {
	r.log.Debugf("received pack data info %s: %d bytes, chunk size %d, chunks %d", pk.UUID, pk.Size, pk.DataChunkSize, pk.ChunkCount)
	if _, ok := r.cache.(*ReplayCache); ok {
		return nil
	}

	wireID, _, _ := strings.Cut(pk.UUID, "_")
	packID, err := uuid.Parse(wireID)
	if err != nil {
		return err
	}
	if pk.DataChunkSize == 0 {
		return fmt.Errorf("resource pack %s has zero chunk size", pk.UUID)
	}
	r.downloadMu.Lock()
	pack, ok := r.downloadingPacks[packID]
	if !ok {
		r.downloadMu.Unlock()
		return fmt.Errorf("unknown or duplicate pack data info for UUID %s", pk.UUID)
	}
	delete(r.downloadingPacks, packID)
	if pack.size != pk.Size {
		r.log.Infof("pack %v had a different size in the ResourcePacksInfo packet than the ResourcePackDataInfo packet", pk.UUID)
		pack.size = pk.Size
	}
	pack.chunkSize = pk.DataChunkSize

	r.awaitingPacks[packID] = &pack
	r.downloadMu.Unlock()
	go func() {
		defer close(pack.done)
		if err := r.downloadResourcePack(pk, &pack); err != nil {
			r.failDownload(fmt.Errorf("download resource pack %s: %w", pack.ID, err))
		}
	}()
	return nil
}

func (r *ResourcePackHandler) OnResourcePackChunkData(pk *packet.ResourcePackChunkData) error {
	if pk.ChunkIndex == 0 {
		r.log.Debugf("first chunk arrived for pack %s: %d bytes", pk.UUID, len(pk.Data))
	}
	if _, ok := r.cache.(*ReplayCache); ok {
		return nil
	}

	wireID, _, _ := strings.Cut(pk.UUID, "_")
	packID, err := uuid.Parse(wireID)
	if err != nil {
		return err
	}
	r.downloadMu.Lock()
	pack := r.awaitingPacks[packID]
	r.downloadMu.Unlock()
	if pack == nil {

		return fmt.Errorf("resource pack chunk data for resource pack that was not being downloaded")
	}

	select {
	case pack.newFrag <- pk:
		return nil
	case <-pack.done:
		return fmt.Errorf("resource pack chunk data for completed resource pack %s", pk.UUID)
	case <-r.ctx.Done():
		return context.Cause(r.ctx)
	case <-r.Server.Context().Done():
		return net.ErrClosed
	}
}

func (r *ResourcePackHandler) OnResourcePackStack(pk *packet.ResourcePackStack) error {

	for _, pack := range pk.TexturePacks {
		id, err := uuid.Parse(pack.UUID)
		if err != nil {
			continue
		}
		if !r.hasPack(id, pack.Version) {
			return fmt.Errorf("texture pack {uuid=%v, version=%v} not downloaded", pack.UUID, pack.Version)
		}
	}

	var addPacks []protocol.StackResourcePack
	for _, p := range r.addedPacks {
		addPacks = append(addPacks, protocol.StackResourcePack{
			UUID:        p.UUID().String(),
			Version:     p.Version(),
			SubPackName: p.Name(),
		})
	}
	pk.TexturePacks = append(addPacks, pk.TexturePacks...)

	r.remoteStack = pk
	close(r.receivedRemoteStack)

	if r.clientDone != nil {
		r.log.Debug("waiting for client to finish downloading")
		select {
		case <-r.clientDone:
		case <-r.ctx.Done():
			return context.Cause(r.ctx)
		case <-r.Server.Context().Done():
			return net.ErrClosed
		}
	}

	r.log.Debug("starting game")
	r.Server.Expect(packet.IDDimensionData, packet.IDStartGame)
	_ = r.Server.WritePacket(&packet.ResourcePackClientResponse{Response: packet.PackResponseCompleted})
	return nil
}

func (r *ResourcePackHandler) OnResourcePackChunkRequest(pk *packet.ResourcePackChunkRequest) error {
	wireID, _, _ := strings.Cut(pk.UUID, "_")
	packID, err := uuid.Parse(wireID)
	if err != nil {
		return err
	}
	r.uploadLock.Lock()
	upload, ok := r.uploads[packID]
	r.uploadLock.Unlock()
	if !ok {
		return fmt.Errorf("client requested an unknown resourcepack chunk %s", packID)
	}
	if upload.currentOffset != uint64(pk.ChunkIndex)*packChunkSize {
		return fmt.Errorf("resource pack chunk request had unexpected chunk index: expected %v, but got %v", upload.currentOffset/packChunkSize, pk.ChunkIndex)
	}
	response := &packet.ResourcePackChunkData{
		UUID:       pk.UUID,
		ChunkIndex: uint32(pk.ChunkIndex),
		DataOffset: upload.currentOffset,
		Data:       make([]byte, packChunkSize),
	}

	upload.currentOffset += packChunkSize

	done := upload.currentOffset >= uint64(upload.Pack.Len())
	if n, err := upload.Pack.ReadAt(response.Data, int64(response.DataOffset)); err != nil {

		if err != io.EOF {
			return fmt.Errorf("error reading resource pack chunk: %v", err)
		}
		response.Data = response.Data[:n]
		done = true
	}
	if err := r.Client.WritePacket(response); err != nil {
		return fmt.Errorf("error writing resource pack chunk data packet: %v", err)
	}

	r.uploadLock.Lock()
	if !done {

		r.Client.Expect(packet.IDResourcePackChunkRequest)
	} else {

		delete(r.uploads, packID)
	}

	if len(r.uploads) == 0 {
		r.Client.Expect(packet.IDResourcePackClientResponse)
	}
	r.uploadLock.Unlock()
	return nil
}

func (r *ResourcePackHandler) processClientRequest(packs []string) error {
	<-r.receivedRemotePackInfo
	var packsFromCache []*resource.Pack
	var addedPacksRequested []*resource.Pack

	if len(packs) == 0 {
		close(r.knowPacksRequestedFromServer)
		return nil
	}

	var contentKeys = make(map[string]string)
	for _, pack := range r.remotePacksInfo.TexturePacks {
		contentKeys[pack.UUID.String()+"_"+pack.Version] = pack.ContentKey
	}

	for _, packUUID := range packs {
		wireID, ver, hasVersion := strings.Cut(packUUID, "_")
		if !hasVersion || ver == "" {
			return fmt.Errorf("resource pack request lacks a version: %q", packUUID)
		}
		id, err := uuid.Parse(wireID)
		if err != nil {
			return err
		}

		if r.cache.Has(id, ver) {
			r.log.Debugf("using %s from cache", packUUID)

			pack, err := r.cache.Get(id, ver)
			if err != nil {
				return err
			}
			if contentKey, ok := contentKeys[packUUID]; ok {
				pack = pack.WithContentKey(contentKey)
			}
			packsFromCache = append(packsFromCache, pack)
			continue
		}

		if idx := slices.IndexFunc(r.addedPacks, func(pack *resource.Pack) bool {
			return pack.UUID().String()+"_"+pack.Version() == packUUID
		}); idx != -1 {
			addedPacksRequested = append(addedPacksRequested, r.addedPacks[idx])
			continue
		}

		if idx := slices.IndexFunc(r.remotePacksInfo.TexturePacks, func(pack protocol.TexturePackInfo) bool {
			return pack.UUID.String()+"_"+pack.Version == packUUID
		}); idx != -1 {
			r.packsRequestedFromServer = append(r.packsRequestedFromServer, packUUID)
			continue
		}

		return fmt.Errorf("could not find resource pack %v", packUUID)
	}

	if len(packsFromCache)+len(r.packsRequestedFromServer)+len(addedPacksRequested) < len(packs) {
		r.log.Errorf("BUG: not enough packs sent to client, client will stall %d + %d  %d", len(packsFromCache), len(r.packsRequestedFromServer), len(packs))
	}

	r.pendingClientPacks = make(map[string]struct{}, len(r.packsRequestedFromServer))
	for _, id := range r.packsRequestedFromServer {
		r.pendingClientPacks[id] = struct{}{}
	}
	r.nextPackToClient = make(chan *resource.Pack, len(packs))
	for _, pack := range addedPacksRequested {
		r.nextPackToClient <- pack
	}
	for _, pack := range packsFromCache {
		r.nextPackToClient <- pack
	}
	if len(r.packsRequestedFromServer) == 0 {
		close(r.nextPackToClient)
	}

	close(r.knowPacksRequestedFromServer)
	return nil
}

func (r *ResourcePackHandler) OnResourcePackClientResponse(pk *packet.ResourcePackClientResponse) error {
	switch pk.Response {
	case packet.PackResponseRefused:

		close(r.clientDone)
		return r.Client.Close()
	case packet.PackResponseSendPacks:
		r.clientHasRequested = true
		if err := r.processClientRequest(pk.PacksToDownload); err != nil {
			return fmt.Errorf("error looking up resource packs to download: %v", err)
		}

		r.uploads = make(map[uuid.UUID]*uploadingPack)
		go func() {
			for {
				select {
				case pack, ok := <-r.nextPackToClient:
					if !ok {
						r.log.Info("finished sending client resource packs")
						return
					}

					r.log.Debugf("next pack %s", pack.Name())
					r.uploadLock.Lock()
					r.uploads[pack.UUID()] = &uploadingPack{
						Pack:          pack,
						currentOffset: 0,
					}
					r.uploadLock.Unlock()

					var packType byte
					switch {
					case pack.HasWorldTemplate():
						packType = packet.ResourcePackTypeWorldTemplate
					case pack.HasTextures() && (pack.HasBehaviours() || pack.HasScripts()):
						packType = packet.ResourcePackTypeAddon
					case !pack.HasTextures() && (pack.HasBehaviours() || pack.HasScripts()):
						packType = packet.ResourcePackTypeBehaviour
					case pack.HasTextures():
						packType = packet.ResourcePackTypeResources
					default:
						packType = packet.ResourcePackTypeSkins
					}

					checksum := pack.Checksum()
					r.Client.Expect(packet.IDResourcePackChunkRequest)
					err := r.Client.WritePacket(&packet.ResourcePackDataInfo{
						UUID:          pack.UUID().String() + "_" + pack.Version(),
						DataChunkSize: packChunkSize,
						ChunkCount:    uint32(pack.DataChunkCount(packChunkSize)),
						Size:          uint64(pack.Len()),
						Hash:          checksum[:],
						PackType:      packType,
					})
					if err != nil {
						r.log.Error(err)
						return
					}

				case <-r.ctx.Done():
					return
				}
			}
		}()

	case packet.PackResponseAllPacksDownloaded:
		if r.allPacksDownloaded {
			return nil
		}
		r.allPacksDownloaded = true
		if !r.clientHasRequested {
			close(r.knowPacksRequestedFromServer)
		}

		r.log.Debug("waiting for remote stack")
		select {
		case <-r.receivedRemoteStack:
		case <-r.ctx.Done():
			return context.Cause(r.ctx)
		}

		if err := r.Client.WritePacket(r.remoteStack); err != nil {
			return fmt.Errorf("error writing resource pack stack packet: %v", err)
		}
	case packet.PackResponseCompleted:
		close(r.clientDone)
		r.Client.SetLoggedIn()
	default:
		return fmt.Errorf("unknown resource pack client response: %v", pk.Response)
	}
	return nil
}

func (r *ResourcePackHandler) GetResourcePacksInfo(texturePacksRequired bool) *packet.ResourcePacksInfo {
	select {
	case <-r.receivedRemotePackInfo:
	case <-r.ctx.Done():
	}

	var pk packet.ResourcePacksInfo
	if r.remotePacksInfo != nil {
		pk.TexturePackRequired = r.remotePacksInfo.TexturePackRequired
		pk.HasAddons = r.remotePacksInfo.HasAddons
		pk.HasScripts = r.remotePacksInfo.HasScripts
		pk.TexturePacks = append(pk.TexturePacks, r.remotePacksInfo.TexturePacks...)
	}

	for _, p := range r.addedPacks {
		pk.TexturePacks = append(pk.TexturePacks, protocol.TexturePackInfo{
			UUID:            p.UUID(),
			Version:         p.Version(),
			Size:            uint64(p.Len()),
			ContentKey:      p.ContentKey(),
			SubPackName:     p.Name(),
			ContentIdentity: p.UUID().String(),
			HasScripts:      false,
			RTXEnabled:      false,
		})
	}

	return &pk
}

func (r *ResourcePackHandler) ResourcePacks() []*resource.Pack {
	select {
	case <-r.receivedRemoteStack:
	case <-r.ctx.Done():
	case <-r.Server.Context().Done():
	}

	r.lockResourcePacks.Lock()
	defer r.lockResourcePacks.Unlock()
	return slices.Clone(r.resourcePacks)
}

func (r *ResourcePackHandler) ActiveResourcePacks() []*resource.Pack {
	return packsInStackOrder(r.ResourcePacks(), r.remoteStack)
}

func packsInStackOrder(packs []*resource.Pack, stack *packet.ResourcePackStack) []*resource.Pack {
	if stack == nil {
		return slices.Clone(packs)
	}
	ordered := make([]*resource.Pack, 0, len(stack.TexturePacks))
	for _, entry := range stack.TexturePacks {
		for _, pack := range packs {
			if strings.EqualFold(pack.UUID().String(), entry.UUID) && pack.Version() == entry.Version {
				ordered = append(ordered, pack)
				break
			}
		}
	}
	return ordered
}

var exemptedPacks = map[string]bool{
	"0fba4063-dba1-4281-9b89-ff9390653530_1.0.0": true,
}

func (r *ResourcePackHandler) hasPack(id uuid.UUID, version string) bool {
	search := id.String() + "_" + version
	if exemptedPacks[search] {

		return true
	}
	if id == uuid.Max {
		return true
	}

	if r.remotePacksInfo != nil {
		if !slices.ContainsFunc(r.remotePacksInfo.TexturePacks, func(pack protocol.TexturePackInfo) bool {
			return pack.UUID == id && pack.Version == version
		}) {
			return true
		}
	}
	return slices.Contains(r.finishedPacks, search)
}
