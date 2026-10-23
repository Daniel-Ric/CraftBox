package minecraft

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/internal"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sandertv/gophertunnel/minecraft/text"
)

type exemptedResourcePack struct {
	uuid    string
	version string
}

var exemptedPacks = []exemptedResourcePack{
	{
		uuid:    "d34cfa4b-2ad1-453d-a0db-668b429a3ea0",
		version: "1.26.40",
	},
	{
		uuid:    "b41c2785-c512-4a49-af56-3a87afd47c57",
		version: "1.21.30",
	},
	{
		uuid:    "a4df0cb3-17be-4163-88d7-fcf7002b935d",
		version: "1.21.20",
	},
	{
		uuid:    "d19adffe-a2e1-4b02-8436-ca4583368c89",
		version: "1.21.10",
	},
	{
		uuid:    "85d5603d-2824-4b21-8044-34f441f4fce1",
		version: "1.21.0",
	},
	{
		uuid:    "e977cd13-0a11-4618-96fb-03dfe9c43608",
		version: "1.20.60",
	},
	{
		uuid:    "0674721c-a0aa-41a1-9ba8-1ed33ea3e7ed",
		version: "1.20.50",
	},
	{
		uuid:    "0fba4063-dba1-4281-9b89-ff9390653530",
		version: "1.0.0",
	},
}

type Conn struct {
	onClientData         func(*Conn)
	ResourcePackHandler  ResourcePackHandler
	prePlayPacketHandler PrePlayPacketHandler
	lastReadTime         time.Time

	once       sync.Once
	ctx        context.Context
	cancelFunc context.CancelCauseFunc

	conn        net.Conn
	log         *slog.Logger
	authEnabled bool

	proto                Protocol
	acceptedProto        []Protocol
	pool                 packet.Pool
	enc                  *packet.Encoder
	dec                  *packet.Decoder
	compression          packet.Compression
	compressionSelector  func(proto Protocol) packet.Compression
	compressionThreshold int
	maxDecompressedLen   int
	readerLimits         bool

	disconnectOnUnknownPacket bool
	disconnectOnInvalidPacket bool

	identityData login.IdentityData
	clientData   login.ClientData

	gameData         GameData
	gameDataReceived atomic.Bool

	privateKey *ecdsa.PrivateKey

	salt              []byte
	disableEncryption bool

	verifier *oidc.IDTokenVerifier

	packets chan *packetData

	deferredPacketMu sync.Mutex

	deferredPackets []*packetData
	readDeadline    <-chan time.Time

	sendMu sync.Mutex

	encMu sync.Mutex

	bufferedSend      [][]byte
	bufferedSendSpare [][]byte
	hdr               *packet.Header

	readyToLogin bool

	authenticated atomic.Bool

	loggedIn bool

	spawn           chan struct{}
	waitingForSpawn atomic.Bool

	expectedIDs atomic.Value

	packMu sync.Mutex

	resourcePacks []*resource.Pack

	texturePacksRequired bool

	forceDisableVibrantVisuals bool
	packQueue                  *resourcePackQueue

	downloadResourcePack func(id uuid.UUID, version string, currentPack, totalPacks int) bool

	fetchResourcePacks func(identityData login.IdentityData, clientData login.ClientData, current []*resource.Pack) []*resource.Pack

	ignoredResourcePacks []exemptedResourcePack

	resourcePackCache ResourcePackCache

	resourcePackDelivery ResourcePackDeliveryConfig

	cacheEnabled bool

	allow func(addr net.Addr, identityData login.IdentityData, clientData login.ClientData) (string, bool)

	packetFunc func(header packet.Header, payload []byte, src, dst net.Addr)

	shieldID atomic.Int32

	additional chan packet.Packet
}

func newConn(netConn net.Conn, key *ecdsa.PrivateKey, log *slog.Logger, proto Protocol, flushRate time.Duration, limits bool) *Conn {
	conn := &Conn{
		enc:          packet.NewEncoder(netConn),
		dec:          packet.NewDecoder(netConn),
		salt:         make([]byte, 16),
		packets:      make(chan *packetData, 8),
		additional:   make(chan packet.Packet, 16),
		spawn:        make(chan struct{}),
		conn:         netConn,
		privateKey:   key,
		log:          log.With("raddr", netConn.RemoteAddr().String()),
		hdr:          &packet.Header{},
		proto:        proto,
		readerLimits: limits,

		resourcePackDelivery: defaultResourcePackDeliveryConfig(),
	}
	if d, ok := netConn.(packet.EncryptionDisabler); ok {
		conn.disableEncryption = d.DisableEncryption()
	}

	if c, ok := netConn.(interface{ Context() context.Context }); ok {
		conn.ctx, conn.cancelFunc = context.WithCancelCause(c.Context())
	} else {
		conn.ctx, conn.cancelFunc = context.WithCancelCause(context.Background())
	}

	if !limits {

		conn.dec.DisableBatchPacketLimit()
	}
	_, _ = rand.Read(conn.salt)

	conn.expectedIDs.Store([]uint32{packet.IDRequestNetworkSettings})

	if flushRate <= 0 {
		return conn
	}
	go func() {
		ticker := time.NewTicker(flushRate)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.Flush(); err != nil {
				_ = conn.close(err)
				return
			}
		}
	}()
	return conn
}

func (conn *Conn) IdentityData() login.IdentityData {
	return conn.identityData
}

func (conn *Conn) ClientData() login.ClientData {
	return conn.clientData
}

func (conn *Conn) Authenticated() bool {
	return conn.IdentityData().XUID != ""
}

func (conn *Conn) GameData() GameData {
	return conn.gameData
}

func (conn *Conn) Proto() Protocol {
	return conn.proto
}

func (conn *Conn) StartGame(data GameData) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return conn.StartGameContext(ctx, data)
}

func (conn *Conn) StartGameTimeout(data GameData, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return conn.StartGameContext(ctx, data)
}

func (conn *Conn) StartGameContext(ctx context.Context, data GameData) error {
	if conn.gameDataReceived.Load() {
		panic("(*Conn).StartGame must only be called on Listener connections")
	}
	if data.WorldName == "" {
		data.WorldName = conn.gameData.WorldName
	}

	conn.gameData = data
	for _, item := range data.Items {
		if item.Name == "minecraft:shield" {
			conn.shieldID.Store(int32(item.RuntimeID))
		}
	}
	conn.waitingForSpawn.Store(true)
	conn.startGame()

	select {
	case <-conn.ctx.Done():
		return conn.closeErr("start game")
	case <-ctx.Done():
		return conn.wrap(ctx.Err(), "start game")
	case <-conn.spawn:

		return nil
	}
}

func (conn *Conn) DoSpawn() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return conn.DoSpawnContext(ctx)
}

func (conn *Conn) DoSpawnTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return conn.DoSpawnContext(ctx)
}

func (conn *Conn) DoSpawnContext(ctx context.Context) error {
	select {
	case <-conn.ctx.Done():
		return conn.closeErr("do spawn")
	case <-ctx.Done():
		return conn.wrap(ctx.Err(), "do spawn")
	case <-conn.spawn:

		return nil
	}
}

func (conn *Conn) WritePacket(pk packet.Packet) error {
	select {
	case <-conn.ctx.Done():
		return conn.closeErr("write packet")
	default:
	}
	conn.sendMu.Lock()
	defer conn.sendMu.Unlock()

	buf := internal.BufferPool.Get().(*bytes.Buffer)
	defer func() {

		buf.Reset()
		internal.BufferPool.Put(buf)
	}()

	for _, converted := range conn.proto.ConvertFromLatest(pk, conn) {
		buf.Reset()
		conn.hdr.PacketID = converted.ID()
		_ = conn.hdr.Write(buf)
		l := buf.Len()

		converted.Marshal(conn.proto.NewWriter(buf, conn.shieldID.Load()))

		if conn.packetFunc != nil {
			conn.packetFunc(*conn.hdr, buf.Bytes()[l:], conn.LocalAddr(), conn.RemoteAddr())
		}
		conn.bufferedSend = append(conn.bufferedSend, append([]byte(nil), buf.Bytes()...))
	}
	return nil
}

func (conn *Conn) ReadPacket() (pk packet.Packet, err error) {
	if len(conn.additional) > 0 {
		return <-conn.additional, nil
	}
	if data, ok := conn.takeDeferredPacket(); ok {
		pk, err := data.decode(conn)
		conn.lastReadTime = data.receivedAt
		if err != nil {
			conn.log.Error("read packet: " + err.Error())
			return conn.ReadPacket()
		}
		if len(pk) == 0 {
			return conn.ReadPacket()
		}
		for _, additional := range pk[1:] {
			conn.additional <- additional
		}
		return pk[0], nil
	}

	select {
	case <-conn.ctx.Done():
		return nil, conn.closeErr("read packet")
	case <-conn.readDeadline:
		return nil, conn.wrap(context.DeadlineExceeded, "read packet")
	case data := <-conn.packets:
		pk, err := data.decode(conn)
		conn.lastReadTime = data.receivedAt
		if err != nil {
			conn.log.Error("read packet: " + err.Error())
			return conn.ReadPacket()
		}
		if len(pk) == 0 {
			return conn.ReadPacket()
		}
		for _, additional := range pk[1:] {
			conn.additional <- additional
		}
		return pk[0], nil
	}
}

func (conn *Conn) ResourcePacks() []*resource.Pack {
	if conn.ResourcePackHandler != nil {
		return conn.ResourcePackHandler.ResourcePacks()
	}
	return conn.resourcePacks
}

func (conn *Conn) Write(b []byte) (n int, err error) {
	conn.sendMu.Lock()
	defer conn.sendMu.Unlock()

	conn.bufferedSend = append(conn.bufferedSend, b)
	return len(b), nil
}

func (conn *Conn) ReadBytes() ([]byte, error) {
	if data, ok := conn.takeDeferredPacket(); ok {
		return data.full, nil
	}
	select {
	case <-conn.ctx.Done():
		return nil, conn.closeErr("read")
	case <-conn.readDeadline:
		return nil, conn.wrap(context.DeadlineExceeded, "read")
	case data := <-conn.packets:
		return data.full, nil
	}
}

func (conn *Conn) Read(b []byte) (n int, err error) {
	if data, ok := conn.takeDeferredPacket(); ok {
		if len(b) < len(data.full) {
			return 0, conn.wrap(errBufferTooSmall, "read")
		}
		return copy(b, data.full), nil
	}
	select {
	case <-conn.ctx.Done():
		return 0, conn.closeErr("read")
	case <-conn.readDeadline:
		return 0, conn.wrap(context.DeadlineExceeded, "read")
	case data := <-conn.packets:
		if len(b) < len(data.full) {
			return 0, conn.wrap(errBufferTooSmall, "read")
		}
		return copy(b, data.full), nil
	}
}

func (conn *Conn) Flush() error {
	if conn.ctx == nil {
		return net.ErrClosed
	}
	select {
	case <-conn.ctx.Done():
		return conn.closeErr("flush")
	default:
	}

	conn.encMu.Lock()
	defer conn.encMu.Unlock()

	conn.sendMu.Lock()
	if len(conn.bufferedSend) == 0 {
		conn.sendMu.Unlock()
		return nil
	}

	toSend := conn.bufferedSend
	conn.bufferedSend = conn.bufferedSendSpare[:0]
	conn.bufferedSendSpare = nil
	conn.sendMu.Unlock()

	if err := conn.enc.Encode(toSend); err != nil && !errors.Is(err, net.ErrClosed) && conn.ctx.Err() == nil {

		panic(fmt.Errorf("error encoding packet batch: %w", err))
	}

	for i := range toSend {
		toSend[i] = nil
	}

	conn.sendMu.Lock()
	conn.bufferedSendSpare = toSend[:0]
	conn.sendMu.Unlock()
	return nil
}

func (conn *Conn) Close() error {
	return conn.close(net.ErrClosed)
}

func (conn *Conn) LocalAddr() net.Addr {
	return conn.conn.LocalAddr()
}

func (conn *Conn) RemoteAddr() net.Addr {
	return conn.conn.RemoteAddr()
}

func (conn *Conn) SetDeadline(t time.Time) error {
	return conn.SetReadDeadline(t)
}

func (conn *Conn) SetReadDeadline(t time.Time) error {
	if t.IsZero() {
		conn.readDeadline = make(chan time.Time)
	} else if t.Before(time.Now()) {
		panic(fmt.Errorf("error setting read deadline: time passed is before time.Now()"))
	} else {
		conn.readDeadline = time.After(time.Until(t))
	}
	return nil
}

func (conn *Conn) SetWriteDeadline(time.Time) error {
	return nil
}

func (conn *Conn) Latency() time.Duration {
	if c, ok := conn.conn.(interface {
		Latency() time.Duration
	}); ok {
		return c.Latency()
	}
	panic(fmt.Sprintf("connection type %T has no Latency() time.Duration method", conn.conn))
}

func (conn *Conn) ClientCacheEnabled() bool {
	return conn.cacheEnabled
}

func (conn *Conn) ChunkRadius() int {
	return int(conn.gameData.ChunkRadius)
}

func (conn *Conn) Context() context.Context {
	return conn.ctx
}

func (conn *Conn) takeDeferredPacket() (*packetData, bool) {
	conn.deferredPacketMu.Lock()
	defer conn.deferredPacketMu.Unlock()

	if len(conn.deferredPackets) == 0 {
		return nil, false
	}
	data := conn.deferredPackets[0]

	conn.deferredPackets[0] = nil
	conn.deferredPackets = conn.deferredPackets[1:]
	return data, true
}

func (conn *Conn) deferPacket(pk *packetData) {
	pk = pk.ensureOwned()
	conn.deferredPacketMu.Lock()
	conn.deferredPackets = append(conn.deferredPackets, pk)
	conn.deferredPacketMu.Unlock()
}

func (conn *Conn) receive(data []byte) error {
	pkData, err := parseData(data, conn)
	if err != nil {
		return err
	}
	if pkData.h.PacketID == packet.IDDisconnect {

		pks, err := pkData.decode(conn)
		if err != nil {
			return err
		}
		_ = conn.close(conn.wrap(DisconnectError(pks[0].(*packet.Disconnect).Message), "receive"))
		return nil
	}
	if conn.loggedIn && !conn.waitingForSpawn.Load() {
		select {
		case <-conn.ctx.Done():
		case previous := <-conn.packets:

			conn.deferPacket(previous)
		default:
		}
		select {
		case <-conn.ctx.Done():
		case conn.packets <- pkData.ensureOwned():
		}
		return nil
	}
	return conn.handle(pkData)
}

func (conn *Conn) handle(pkData *packetData) error {
	for _, id := range conn.expectedIDs.Load().([]uint32) {
		if id == pkData.h.PacketID {

			pks, err := pkData.decode(conn)
			if err != nil {
				return err
			}
			return conn.handleMultiple(pks)
		}
	}
	if conn.prePlayPacketHandler != nil {
		copy := &packetData{h: pkData.h, payload: bytes.NewBuffer(bytes.Clone(pkData.payload.Bytes())), owned: true}
		packets, err := copy.decode(conn)
		if err != nil {
			return err
		}
		var unhandled []packet.Packet
		for _, pk := range packets {
			handled, err := conn.prePlayPacketHandler(conn, pk, pkData.receivedAt)
			if err != nil {
				return err
			}
			if !handled {
				unhandled = append(unhandled, pk)
			}
		}
		if len(unhandled) != 0 {
			retained := pkData.ensureOwned()
			retained.decoded = unhandled
			conn.deferPacket(retained)
		}
		return nil
	}

	conn.deferPacket(pkData)
	return nil
}

func (conn *Conn) handleMultiple(pks []packet.Packet) error {
	var err error
	for _, pk := range pks {
		if e := conn.handlePacket(pk); e != nil {
			err = fmt.Errorf("handle %T: %w", pk, e)
		}
	}
	return err
}

func (conn *Conn) handlePacket(pk packet.Packet) error {
	defer func() {
		_ = conn.Flush()
	}()
	if r := conn.ResourcePackHandler; r != nil {
		switch pk := pk.(type) {
		case *packet.ResourcePacksInfo:
			return r.OnResourcePacksInfo(pk)
		case *packet.ResourcePackClientResponse:
			return r.OnResourcePackClientResponse(pk)
		case *packet.ResourcePackDataInfo:
			return r.OnResourcePackDataInfo(pk)
		case *packet.ResourcePackChunkRequest:
			return r.OnResourcePackChunkRequest(pk)
		case *packet.ResourcePackChunkData:
			return r.OnResourcePackChunkData(pk)
		case *packet.ResourcePackStack:
			return r.OnResourcePackStack(pk)
		}
	}
	switch pk := pk.(type) {

	case *packet.RequestNetworkSettings:
		return conn.handleRequestNetworkSettings(pk)
	case *packet.Login:
		return conn.handleLogin(pk)
	case *packet.ClientToServerHandshake:
		return conn.handleClientToServerHandshake()
	case *packet.ClientCacheStatus:
		return conn.handleClientCacheStatus(pk)
	case *packet.ResourcePackClientResponse:
		return conn.handleResourcePackClientResponse(pk)
	case *packet.ResourcePackChunkRequest:
		return conn.handleResourcePackChunkRequest(pk)
	case *packet.RequestChunkRadius:
		return conn.handleRequestChunkRadius(pk)
	case *packet.SetLocalPlayerAsInitialised:
		return conn.handleSetLocalPlayerAsInitialised(pk)

	case *packet.NetworkSettings:
		return conn.handleNetworkSettings(pk)
	case *packet.ServerToClientHandshake:
		return conn.handleServerToClientHandshake(pk)
	case *packet.PlayStatus:
		return conn.handlePlayStatus(pk)
	case *packet.ResourcePacksInfo:
		return conn.handleResourcePacksInfo(pk)
	case *packet.ResourcePackDataInfo:
		return conn.handleResourcePackDataInfo(pk)
	case *packet.ResourcePackChunkData:
		return conn.handleResourcePackChunkData(pk)
	case *packet.ResourcePackStack:
		return conn.handleResourcePackStack(pk)
	case *packet.StartGame:
		return conn.handleStartGame(pk)
	case *packet.ItemRegistry:
		return conn.handleItemRegistry(pk)
	case *packet.ChunkRadiusUpdated:
		return conn.handleChunkRadiusUpdated(pk)
	case *packet.DimensionData:
		return conn.handleDimensionData(pk)
	}
	return nil
}

func (conn *Conn) handleRequestNetworkSettings(pk *packet.RequestNetworkSettings) error {
	found := false
	for _, pro := range conn.acceptedProto {
		if pro.ID() == pk.ClientProtocol {
			conn.proto = pro
			conn.pool = pro.Packets(true)
			found = true
			break
		}
	}
	if !found {
		status := packet.PlayStatusLoginFailedClient
		if pk.ClientProtocol > protocol.CurrentProtocol {

			status = packet.PlayStatusLoginFailedServer
		}
		_ = conn.WritePacket(&packet.PlayStatus{Status: status})
		return fmt.Errorf("incompatible protocol version: expected %v, got %v", protocol.CurrentProtocol, pk.ClientProtocol)
	}

	if conn.compressionSelector != nil {
		if c := conn.compressionSelector(conn.proto); c != nil {
			conn.compression = c
		}
	}

	conn.expect(packet.IDLogin)
	if err := conn.WritePacket(&packet.NetworkSettings{
		CompressionThreshold: uint16(conn.compressionThreshold),
		CompressionAlgorithm: conn.compression.EncodeCompression(),
	}); err != nil {
		return fmt.Errorf("send NetworkSettings: %w", err)
	}
	_ = conn.Flush()
	conn.encMu.Lock()
	conn.enc.EnableCompression(conn.compression, conn.compressionThreshold)
	conn.encMu.Unlock()
	conn.dec.EnableCompression(conn.compression, conn.maxDecompressedLen)
	return nil
}

func (conn *Conn) handleNetworkSettings(pk *packet.NetworkSettings) error {
	alg, ok := packet.CompressionByID(pk.CompressionAlgorithm)
	if !ok {
		return fmt.Errorf("unknown compression algorithm %v", pk.CompressionAlgorithm)
	}
	conn.encMu.Lock()
	conn.enc.EnableCompression(alg, int(pk.CompressionThreshold))
	conn.encMu.Unlock()
	conn.dec.EnableCompression(alg, conn.maxDecompressedLen)
	conn.readyToLogin = true
	return nil
}

func (conn *Conn) handleLogin(pk *packet.Login) error {
	var (
		err        error
		authResult login.AuthResult
	)
	conn.identityData, conn.clientData, authResult, err = login.Parse(pk.ConnectionRequest, conn.verifier)
	if err != nil {
		return fmt.Errorf("parse login request: %w", err)
	}
	if conn.onClientData != nil {
		conn.onClientData(conn)
	}

	if !authResult.XBOXLiveAuthenticated && conn.authEnabled {
		_ = conn.WritePacket(&packet.Disconnect{Message: text.Colourf("<red>You must be logged in with XBOX Live to join.</red>")})
		return fmt.Errorf("client was not authenticated to XBOX Live")
	}
	if v, ok := conn.conn.(publicKeyVerifierConn); ok {
		if err := v.VerifyPublicKey(authResult.PublicKey); err != nil {
			_ = conn.WritePacket(&packet.Disconnect{Reason: packet.DisconnectReasonNotAuthenticated})
			return fmt.Errorf("verify identity public key: %w", err)
		}
	} else if pkc, ok := conn.conn.(publicKeyConn); ok {
		if pub := pkc.PublicKey(); pub != nil && !authResult.PublicKey.Equal(pub) {
			_ = conn.WritePacket(&packet.Disconnect{Reason: packet.DisconnectReasonNotAuthenticated})
			return fmt.Errorf("identity public key mismatch: %s != %s", login.MarshalPublicKey(authResult.PublicKey), login.MarshalPublicKey(pub))
		}
	}
	if conn.allow != nil {
		if reason, ok := conn.allow(conn.RemoteAddr(), conn.identityData, conn.clientData); !ok {
			_ = conn.WritePacket(&packet.Disconnect{Reason: packet.DisconnectReasonKicked, Message: reason})
			return conn.Close()
		}
	}
	if conn.disableEncryption {
		return conn.handleClientToServerHandshake()
	}

	conn.expect(packet.IDClientToServerHandshake)
	if err := conn.enableEncryption(authResult.PublicKey); err != nil {
		return fmt.Errorf("enable encryption: %w", err)
	}
	return nil
}

type publicKeyConn interface {
	PublicKey() *ecdsa.PublicKey
}

type publicKeyVerifierConn interface {
	VerifyPublicKey(pub *ecdsa.PublicKey) error
}

func (conn *Conn) handleClientToServerHandshake() error {

	conn.authenticated.Store(true)

	conn.expect(packet.IDResourcePackClientResponse, packet.IDClientCacheStatus)
	if err := conn.WritePacket(&packet.PlayStatus{Status: packet.PlayStatusLoginSuccess}); err != nil {
		return fmt.Errorf("send PlayStatus (Status=LoginSuccess): %w", err)
	}

	if conn.fetchResourcePacks != nil {
		conn.resourcePacks = conn.fetchResourcePacks(conn.identityData, conn.clientData, slices.Clone(conn.resourcePacks))
	}
	pk := &packet.ResourcePacksInfo{TexturePackRequired: conn.texturePacksRequired, ForceDisableVibrantVisuals: conn.forceDisableVibrantVisuals}
	for _, pack := range conn.resourcePacks {
		texturePack := protocol.TexturePackInfo{
			UUID:        pack.UUID(),
			Version:     pack.Version(),
			Size:        uint64(pack.Len()),
			DownloadURL: pack.DownloadURL(),
		}
		if pack.Encrypted() {
			texturePack.ContentKey = pack.ContentKey()
			texturePack.ContentIdentity = pack.Manifest().Header.UUID.String()
		}
		pk.TexturePacks = append(pk.TexturePacks, texturePack)
	}

	if conn.ResourcePackHandler != nil {
		pk = conn.ResourcePackHandler.GetResourcePacksInfo(conn.texturePacksRequired)
	}
	if err := conn.WritePacket(pk); err != nil {
		return fmt.Errorf("send ResourcePacksInfo: %w", err)
	}
	return nil
}

type saltClaims struct {
	Salt string `json:"salt"`
}

func (conn *Conn) handleServerToClientHandshake(pk *packet.ServerToClientHandshake) error {
	tok, err := jwt.ParseSigned(string(pk.JWT), []jose.SignatureAlgorithm{jose.ES384})
	if err != nil {
		return fmt.Errorf("parse server token: %w", err)
	}
	raw := tok.Headers[0].ExtraHeaders["x5u"]
	kStr, _ := raw.(string)

	pub := new(ecdsa.PublicKey)
	if err := login.ParsePublicKey(kStr, pub); err != nil {
		return fmt.Errorf("parse server public key: %w", err)
	}

	var c saltClaims
	if err := tok.Claims(pub, &c); err != nil {
		return fmt.Errorf("verify claims: %w", err)
	}
	c.Salt = strings.TrimRight(c.Salt, "=")
	salt, err := base64.RawStdEncoding.DecodeString(c.Salt)
	if err != nil {
		return fmt.Errorf("decode ServerToClientHandshake salt: %w", err)
	}

	keyBytes, err := conn.encryptionKey(salt, pub)
	if err != nil {
		return fmt.Errorf("derive encryption key: %w", err)
	}

	conn.encMu.Lock()
	conn.enc.EnableEncryption(keyBytes)
	conn.encMu.Unlock()
	conn.dec.EnableEncryption(keyBytes)

	_ = conn.WritePacket(&packet.ClientToServerHandshake{})
	return nil
}

func (conn *Conn) handleClientCacheStatus(pk *packet.ClientCacheStatus) error {
	conn.cacheEnabled = pk.Enabled
	return nil
}

func (conn *Conn) handleResourcePacksInfo(pk *packet.ResourcePacksInfo) error {

	totalPacks := len(pk.TexturePacks)
	conn.packQueue = &resourcePackQueue{
		packAmount:       totalPacks,
		downloadingPacks: make(map[string]downloadingPack),
		awaitingPacks:    make(map[string]*downloadingPack),
	}
	packsToDownload := make([]string, 0, totalPacks)

	for index, pack := range pk.TexturePacks {
		id := pack.UUID.String()
		if _, ok := conn.packQueue.downloadingPacks[id]; ok {
			conn.log.Warn("handle ResourcePacksInfo: duplicate texture pack", "UUID", pack.UUID)
			conn.packQueue.packAmount--
			continue
		}
		if conn.downloadResourcePack != nil && !conn.downloadResourcePack(uuid.MustParse(id), pack.Version, index, totalPacks) {
			conn.ignoredResourcePacks = append(conn.ignoredResourcePacks, exemptedResourcePack{
				uuid:    id,
				version: pack.Version,
			})
			conn.packQueue.packAmount--
			continue
		}

		cacheKey := ResourcePackCacheKey{
			UUID:    pack.UUID,
			Version: pack.Version,
			Size:    pack.Size,
		}
		if conn.resourcePackCache != nil {
			cachedPack, err := conn.resourcePackCache.Load(conn.ctx, cacheKey)
			switch {
			case err != nil:
				conn.log.Warn("handle ResourcePacksInfo: failed to load resource pack from cache", "UUID", pack.UUID, "version", pack.Version, "err", err)
			case cachedPack != nil && !cacheKey.Matches(cachedPack):
				conn.log.Warn("handle ResourcePacksInfo: cached resource pack did not match advertised pack", "UUID", pack.UUID, "version", pack.Version, "cached_UUID", cachedPack.UUID(), "cached_version", cachedPack.Version(), "cached_size", cachedPack.Len())
			case cachedPack != nil:
				conn.resourcePacks = append(conn.resourcePacks, cachedPack.WithContentKey(pack.ContentKey))
				conn.packQueue.packAmount--
				continue
			}
		}

		packsToDownload = append(packsToDownload, id+"_"+pack.Version)
		conn.packQueue.downloadingPacks[id] = downloadingPack{
			size:       pack.Size,
			buf:        bytes.NewBuffer(make([]byte, 0, pack.Size)),
			newFrag:    make(chan []byte),
			contentKey: pack.ContentKey,
			cacheKey:   cacheKey,
		}
	}

	if len(packsToDownload) != 0 {
		conn.expect(packet.IDResourcePackDataInfo, packet.IDResourcePackChunkData, packet.IDPlayStatus)
		_ = conn.WritePacket(&packet.ResourcePackClientResponse{
			Response:        packet.PackResponseSendPacks,
			PacksToDownload: packsToDownload,
		})
		return nil
	}
	conn.expect(packet.IDResourcePackStack, packet.IDPlayStatus)

	_ = conn.WritePacket(&packet.ResourcePackClientResponse{Response: packet.PackResponseAllPacksDownloaded})
	return nil
}

func (conn *Conn) storeResourcePack(key ResourcePackCacheKey, pack *resource.Pack) {
	if conn.resourcePackCache == nil || !key.Matches(pack) {

		return
	}
	if err := conn.resourcePackCache.Store(conn.ctx, key, pack); err != nil {
		conn.log.Warn("failed to store resource pack in cache", "UUID", pack.UUID(), "version", pack.Version(), "err", err)
	}
}

func (conn *Conn) handleResourcePackStack(pk *packet.ResourcePackStack) error {

	for _, pack := range pk.TexturePacks {
		if !conn.hasPack(pack.UUID, pack.Version, false) {
			return fmt.Errorf("texture pack (UUID=%v, version=%v) not downloaded", pack.UUID, pack.Version)
		}
	}
	conn.expect(packet.IDDimensionData, packet.IDStartGame)
	_ = conn.WritePacket(&packet.ResourcePackClientResponse{Response: packet.PackResponseCompleted})
	return nil
}

func (conn *Conn) hasPack(uuid string, version string, hasBehaviours bool) bool {
	for _, exempted := range exemptedPacks {
		if exempted.uuid == uuid && exempted.version == version {

			return true
		}
	}
	conn.packMu.Lock()
	defer conn.packMu.Unlock()

	for _, ignored := range conn.ignoredResourcePacks {
		if ignored.uuid == uuid && ignored.version == version {
			return true
		}
	}
	for _, pack := range conn.resourcePacks {
		if pack.UUID().String() == uuid && pack.Version() == version && pack.HasBehaviours() == hasBehaviours {
			return true
		}
	}
	return false
}

func (conn *Conn) handleResourcePackClientResponse(pk *packet.ResourcePackClientResponse) error {
	switch pk.Response {
	case packet.PackResponseRefused:

		return conn.close(conn.closeErr("resource pack refused"))
	case packet.PackResponseSendPacks:
		packs := pk.PacksToDownload
		if len(packs) == 0 {
			break
		}
		conn.packQueue = &resourcePackQueue{
			packs:     conn.resourcePacks,
			chunkSize: conn.resourcePackDelivery.ChunkSize,
		}
		if err := conn.packQueue.Request(packs); err != nil {
			return fmt.Errorf("lookup resource packs by UUID: %w", err)
		}

		if err := conn.nextResourcePackDownload(); err != nil {
			return err
		}
	case packet.PackResponseAllPacksDownloaded:
		pk := &packet.ResourcePackStack{BaseGameVersion: protocol.CurrentVersion, Experiments: []protocol.ExperimentData{{Name: "cameras", Enabled: true}}}
		for _, pack := range conn.resourcePacks {
			resourcePack := protocol.StackResourcePack{UUID: pack.UUID().String(), Version: pack.Version()}
			pk.TexturePacks = append(pk.TexturePacks, resourcePack)
		}
		for _, exempted := range exemptedPacks {
			pk.TexturePacks = append(pk.TexturePacks, protocol.StackResourcePack{
				UUID:    exempted.uuid,
				Version: exempted.version,
			})
		}
		if err := conn.WritePacket(pk); err != nil {
			return fmt.Errorf("send ResourcePackStack: %w", err)
		}
	case packet.PackResponseCompleted:
		conn.loggedIn = true
	default:
		return fmt.Errorf("unknown ResourcePackClientResponse response type %v", pk.Response)
	}
	return nil
}

func (conn *Conn) startGame() {

	conn.expect(packet.IDRequestChunkRadius, packet.IDSetLocalPlayerAsInitialised)
	data := conn.gameData
	if len(data.Dimensions) > 0 {
		_ = conn.WritePacket(&packet.DimensionData{Definitions: data.Dimensions})
	}
	_ = conn.WritePacket(&packet.JigsawStructureData{
		StructureData: map[string]any{
			"processors":     make([]map[string]any, 0),
			"template_pools": make([]map[string]any, 0),
			"jigsaws":        make([]map[string]any, 0),
			"structure_sets": make([]map[string]any, 0),
		},
	})
	_ = conn.WritePacket(&packet.VoxelShapes{})
	_ = conn.WritePacket(&packet.StartGame{
		Difficulty:                   data.Difficulty,
		EntityUniqueID:               data.EntityUniqueID,
		EntityRuntimeID:              data.EntityRuntimeID,
		PlayerGameMode:               data.PlayerGameMode,
		PlayerPosition:               data.PlayerPosition,
		Pitch:                        data.Pitch,
		Yaw:                          data.Yaw,
		WorldSeed:                    data.WorldSeed,
		Dimension:                    data.Dimension,
		WorldSpawn:                   data.WorldSpawn,
		EditorWorldType:              data.EditorWorldType,
		CreatedInEditor:              data.CreatedInEditor,
		ExportedFromEditor:           data.ExportedFromEditor,
		PersonaDisabled:              data.PersonaDisabled,
		CustomSkinsDisabled:          data.CustomSkinsDisabled,
		EmoteChatMuted:               data.EmoteChatMuted,
		GameRules:                    data.GameRules,
		Time:                         data.Time,
		DayCycleLockTime:             data.DayCycleLockTime,
		Blocks:                       data.CustomBlocks,
		AchievementsDisabled:         true,
		Generator:                    1,
		EducationFeaturesEnabled:     true,
		MultiPlayerGame:              true,
		MultiPlayerCorrelationID:     uuid.Must(uuid.NewRandom()).String(),
		CommandsEnabled:              true,
		WorldName:                    data.WorldName,
		LANBroadcastEnabled:          true,
		PlayerMovementSettings:       data.PlayerMovementSettings,
		WorldGameMode:                data.WorldGameMode,
		Hardcore:                     data.Hardcore,
		XBLBroadcastMode:             data.XBLBroadcastMode,
		ServerAuthoritativeInventory: data.ServerAuthoritativeInventory,
		PlayerPermissions:            data.PlayerPermissions,
		Experiments:                  data.Experiments,
		ClientSideGeneration:         data.ClientSideGeneration,
		ChatRestrictionLevel:         data.ChatRestrictionLevel,
		DisablePlayerInteractions:    data.DisablePlayerInteractions,
		BaseGameVersion:              data.BaseGameVersion,
		GameVersion:                  protocol.CurrentVersion,
		UseBlockNetworkIDHashes:      data.UseBlockNetworkIDHashes,
		PropertyData:                 data.PropertyData,
	})
	_ = conn.WritePacket(&packet.ItemRegistry{Items: data.Items})
	_ = conn.Flush()
}

func (conn *Conn) nextResourcePackDownload() error {
	pk, ok := conn.packQueue.NextPack()
	if !ok {
		return fmt.Errorf("no resource packs to download")
	}
	if err := conn.WritePacket(pk); err != nil {
		return fmt.Errorf("send ResourcePackDataInfo: %w", err)
	}

	conn.expect(packet.IDResourcePackChunkRequest)
	return nil
}

func (conn *Conn) handleResourcePackDataInfo(pk *packet.ResourcePackDataInfo) error {
	id := strings.Split(pk.UUID, "_")[0]

	pack, ok := conn.packQueue.downloadingPacks[id]
	if !ok {

		return fmt.Errorf("handle ResourcePackDataInfo: unknown pack (UUID=%v)", id)
	}
	if pack.size != pk.Size {

		conn.log.Warn("handle ResourcePackDataInfo: pack had a different size in ResourcePacksInfo than in ResourcePackDataInfo", "UUID", id)
		pack.size = pk.Size
	}

	delete(conn.packQueue.downloadingPacks, id)
	conn.packQueue.awaitingPacks[id] = &pack

	pack.chunkSize = pk.DataChunkSize

	chunkCount := int32(pk.Size / uint64(pk.DataChunkSize))
	if pk.Size%uint64(pk.DataChunkSize) != 0 {
		chunkCount++
	}

	idCopy := pk.UUID
	go func() {
		for i := int32(0); i < chunkCount; i++ {
			_ = conn.WritePacket(&packet.ResourcePackChunkRequest{
				UUID:       idCopy,
				ChunkIndex: i,
			})
			select {
			case <-conn.ctx.Done():
				return
			case frag := <-pack.newFrag:

				_, _ = pack.buf.Write(frag)
			}
		}
		conn.packMu.Lock()
		if pack.buf.Len() != int(pack.size) {
			conn.log.Error(fmt.Sprintf("download resource pack: incorrect resource pack size: expected %v, got %v", pack.size, pack.buf.Len()), "UUID", id)
			conn.packMu.Unlock()
			return
		}

		newPack, err := resource.Read(pack.buf)
		if err != nil {
			conn.log.Error("download resource pack: invalid full resource pack data: "+err.Error(), "UUID", id)
			conn.packMu.Unlock()
			return
		}
		newPack = newPack.WithContentKey(pack.contentKey)
		conn.packQueue.packAmount--

		conn.resourcePacks = append(conn.resourcePacks, newPack)
		packAmount := conn.packQueue.packAmount
		conn.packMu.Unlock()

		if packAmount == 0 {
			conn.expect(packet.IDResourcePackStack, packet.IDPlayStatus)
			_ = conn.WritePacket(&packet.ResourcePackClientResponse{Response: packet.PackResponseAllPacksDownloaded})
		}
		conn.storeResourcePack(pack.cacheKey, newPack)
	}()
	return nil
}

func (conn *Conn) handleResourcePackChunkData(pk *packet.ResourcePackChunkData) error {
	pk.UUID = strings.Split(pk.UUID, "_")[0]
	pack, ok := conn.packQueue.awaitingPacks[pk.UUID]
	if !ok {

		return fmt.Errorf("chunk data for resource pack that was not being downloaded")
	}
	lastData := pack.buf.Len()+int(pack.chunkSize) >= int(pack.size)
	if !lastData && uint32(len(pk.Data)) != pack.chunkSize {

		return fmt.Errorf("expected chunk size %v, got %v", pack.chunkSize, len(pk.Data))
	}
	if pk.ChunkIndex != pack.expectedIndex {
		return fmt.Errorf("expected chunk index %v, got %v", pack.expectedIndex, pk.ChunkIndex)
	}
	pack.expectedIndex++
	pack.newFrag <- pk.Data
	return nil
}

func (conn *Conn) handleResourcePackChunkRequest(pk *packet.ResourcePackChunkRequest) error {
	current := conn.packQueue.currentPack
	chunkSize := uint64(conn.packQueue.chunkSize)
	uuid, _, _ := strings.Cut(pk.UUID, "_")
	if current.UUID().String() != uuid {
		return fmt.Errorf("expected pack UUID %v, but got %v", current.UUID(), pk.UUID)
	}
	if conn.packQueue.currentOffset != uint64(pk.ChunkIndex)*chunkSize {
		return fmt.Errorf("expected chunk index %v, but got %v", conn.packQueue.currentOffset/chunkSize, pk.ChunkIndex)
	}
	response := &packet.ResourcePackChunkData{
		UUID:       pk.UUID,
		ChunkIndex: uint32(pk.ChunkIndex),
		DataOffset: conn.packQueue.currentOffset,
		Data:       make([]byte, chunkSize),
	}
	conn.packQueue.currentOffset += chunkSize

	if n, err := current.ReadAt(response.Data, int64(response.DataOffset)); err != nil {

		if err != io.EOF {
			return fmt.Errorf("read resource pack chunk: %w", err)
		}
		response.Data = response.Data[:n]
	}
	if err := conn.WritePacket(response); err != nil {
		return fmt.Errorf("send ResourcePackChunkData: %w", err)
	}
	if err := conn.Flush(); err != nil {
		return fmt.Errorf("flush ResourcePackChunkData: %w", err)
	}

	lastChunk := response.DataOffset+uint64(len(response.Data)) >= uint64(current.Len())
	if lastChunk {
		if !conn.packQueue.AllDownloaded() {
			_ = conn.nextResourcePackDownload()
		} else {
			conn.expect(packet.IDResourcePackClientResponse)
		}
	}
	if err := waitResourcePackChunkSendDelay(conn.ctx, conn.resourcePackDelivery.ChunkSendDelay); err != nil {
		return err
	}

	return nil
}

func waitResourcePackChunkSendDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (conn *Conn) handleDimensionData(pk *packet.DimensionData) error {
	conn.gameData.Dimensions = pk.Definitions
	return nil
}

func (conn *Conn) handleStartGame(pk *packet.StartGame) error {
	conn.gameData = GameData{
		Difficulty:                   pk.Difficulty,
		WorldName:                    pk.WorldName,
		WorldSeed:                    pk.WorldSeed,
		EntityUniqueID:               pk.EntityUniqueID,
		EntityRuntimeID:              pk.EntityRuntimeID,
		PlayerGameMode:               pk.PlayerGameMode,
		BaseGameVersion:              pk.BaseGameVersion,
		PlayerPosition:               pk.PlayerPosition,
		Pitch:                        pk.Pitch,
		Yaw:                          pk.Yaw,
		Dimension:                    pk.Dimension,
		WorldSpawn:                   pk.WorldSpawn,
		EditorWorldType:              pk.EditorWorldType,
		CreatedInEditor:              pk.CreatedInEditor,
		ExportedFromEditor:           pk.ExportedFromEditor,
		PersonaDisabled:              pk.PersonaDisabled,
		CustomSkinsDisabled:          pk.CustomSkinsDisabled,
		EmoteChatMuted:               pk.EmoteChatMuted,
		GameRules:                    pk.GameRules,
		Time:                         pk.Time,
		DayCycleLockTime:             pk.DayCycleLockTime,
		ServerBlockStateChecksum:     pk.ServerBlockStateChecksum,
		CustomBlocks:                 pk.Blocks,
		PlayerMovementSettings:       pk.PlayerMovementSettings,
		WorldGameMode:                pk.WorldGameMode,
		Hardcore:                     pk.Hardcore,
		XBLBroadcastMode:             pk.XBLBroadcastMode,
		ServerAuthoritativeInventory: pk.ServerAuthoritativeInventory,
		PlayerPermissions:            pk.PlayerPermissions,
		ChatRestrictionLevel:         pk.ChatRestrictionLevel,
		DisablePlayerInteractions:    pk.DisablePlayerInteractions,
		ClientSideGeneration:         pk.ClientSideGeneration,
		Experiments:                  pk.Experiments,
		UseBlockNetworkIDHashes:      pk.UseBlockNetworkIDHashes,
		PropertyData:                 pk.PropertyData,
		Dimensions:                   conn.gameData.Dimensions,
	}
	conn.expect(packet.IDItemRegistry)
	return nil
}

func (conn *Conn) handleItemRegistry(pk *packet.ItemRegistry) error {
	conn.gameData.Items = pk.Items
	for _, item := range pk.Items {
		if item.Name == "minecraft:shield" {
			conn.shieldID.Store(int32(item.RuntimeID))
		}
	}

	_ = conn.WritePacket(&packet.RequestChunkRadius{ChunkRadius: 16, MaxChunkRadius: 16})
	conn.expect(packet.IDChunkRadiusUpdated, packet.IDPlayStatus)
	return nil
}

func (conn *Conn) handleRequestChunkRadius(pk *packet.RequestChunkRadius) error {
	if pk.ChunkRadius < 1 {
		return fmt.Errorf("expected chunk radius of at least 1, got %v", pk.ChunkRadius)
	}
	conn.expect(packet.IDSetLocalPlayerAsInitialised)
	radius := pk.ChunkRadius
	if r := conn.gameData.ChunkRadius; r != 0 {
		radius = r
	}
	_ = conn.WritePacket(&packet.ChunkRadiusUpdated{ChunkRadius: radius})
	conn.gameData.ChunkRadius = pk.ChunkRadius
	_ = conn.WritePacket(&packet.PlayStatus{Status: packet.PlayStatusPlayerSpawn})
	_ = conn.WritePacket(&packet.CreativeContent{})
	return nil
}

func (conn *Conn) handleChunkRadiusUpdated(pk *packet.ChunkRadiusUpdated) error {
	if pk.ChunkRadius < 1 {
		return fmt.Errorf("expected chunk radius of at least 1, got %v", pk.ChunkRadius)
	}

	conn.expect(packet.IDPlayStatus, packet.IDResourcePacksInfo)

	conn.gameData.ChunkRadius = pk.ChunkRadius
	conn.gameDataReceived.Store(true)

	conn.tryFinaliseClientConn()
	return nil
}

func (conn *Conn) handleSetLocalPlayerAsInitialised(pk *packet.SetLocalPlayerAsInitialised) error {
	if pk.EntityRuntimeID != conn.gameData.EntityRuntimeID {
		return fmt.Errorf("entity runtime ID mismatch: expected %v (from StartGame), got %v", conn.gameData.EntityRuntimeID, pk.EntityRuntimeID)
	}
	if conn.waitingForSpawn.CompareAndSwap(true, false) {
		close(conn.spawn)
	}
	return nil
}

func (conn *Conn) handlePlayStatus(pk *packet.PlayStatus) error {
	switch pk.Status {
	case packet.PlayStatusLoginSuccess:
		if err := conn.WritePacket(&packet.ClientCacheStatus{Enabled: conn.cacheEnabled}); err != nil {
			return fmt.Errorf("send ClientCacheStatus: %w", err)
		}

		return conn.Flush()
	case packet.PlayStatusLoginFailedClient:
		_ = conn.close(conn.closeErr("client outdated"))
		return fmt.Errorf("client outdated")
	case packet.PlayStatusLoginFailedServer:
		_ = conn.close(conn.closeErr("server outdated"))
		return fmt.Errorf("server outdated")
	case packet.PlayStatusPlayerSpawn:

		conn.waitingForSpawn.Store(true)
		conn.tryFinaliseClientConn()
		return nil
	case packet.PlayStatusLoginFailedInvalidTenant:
		_ = conn.close(conn.closeErr("invalid edu edition game owner"))
		return fmt.Errorf("invalid edu edition game owner")
	case packet.PlayStatusLoginFailedVanillaEdu:
		_ = conn.close(conn.closeErr("cannot join an edu edition game on vanilla"))
		return fmt.Errorf("cannot join an edu edition game on vanilla")
	case packet.PlayStatusLoginFailedEduVanilla:
		_ = conn.close(conn.closeErr("cannot join a vanilla game on edu edition"))
		return fmt.Errorf("cannot join a vanilla game on edu edition")
	case packet.PlayStatusLoginFailedServerFull:
		_ = conn.close(conn.closeErr("server full"))
		return fmt.Errorf("server full")
	case packet.PlayStatusLoginFailedEditorVanilla:
		_ = conn.close(conn.closeErr("cannot join a vanilla game on editor"))
		return fmt.Errorf("cannot join a vanilla game on editor")
	case packet.PlayStatusLoginFailedVanillaEditor:
		_ = conn.close(conn.closeErr("cannot join an editor game on vanilla"))
		return fmt.Errorf("cannot join an editor game on vanilla")
	default:
		return fmt.Errorf("unknown play status %v", pk.Status)
	}
}

func (conn *Conn) tryFinaliseClientConn() {
	if conn.waitingForSpawn.Load() && conn.gameDataReceived.Load() {
		conn.waitingForSpawn.Store(false)
		conn.gameDataReceived.Store(false)

		close(conn.spawn)
		conn.loggedIn = true
		_ = conn.WritePacket(&packet.SetLocalPlayerAsInitialised{EntityRuntimeID: conn.gameData.EntityRuntimeID})
	}
}

func (conn *Conn) enableEncryption(clientPublicKey *ecdsa.PublicKey) error {
	signer, _ := jose.NewSigner(jose.SigningKey{Key: conn.privateKey, Algorithm: jose.ES384}, &jose.SignerOptions{
		ExtraHeaders: map[jose.HeaderKey]any{"x5u": login.MarshalPublicKey(&conn.privateKey.PublicKey)},
	})

	serverJWT, err := jwt.Signed(signer).Claims(saltClaims{Salt: base64.RawStdEncoding.EncodeToString(conn.salt)}).Serialize()
	if err != nil {
		return fmt.Errorf("compact serialise server JWT: %w", err)
	}
	if err := conn.WritePacket(&packet.ServerToClientHandshake{JWT: []byte(serverJWT)}); err != nil {
		return fmt.Errorf("send ServerToClientHandshake: %w", err)
	}

	_ = conn.Flush()

	keyBytes, err := conn.encryptionKey(conn.salt, clientPublicKey)
	if err != nil {
		return fmt.Errorf("derive encryption key: %w", err)
	}

	conn.encMu.Lock()
	conn.enc.EnableEncryption(keyBytes)
	conn.encMu.Unlock()
	conn.dec.EnableEncryption(keyBytes)

	return nil
}

func (conn *Conn) encryptionKey(salt []byte, pub *ecdsa.PublicKey) ([32]byte, error) {
	privateKey, err := conn.privateKey.ECDH()
	if err != nil {
		return [32]byte{}, fmt.Errorf("convert private key to ECDH: %w", err)
	}
	publicKey, err := pub.ECDH()
	if err != nil {
		return [32]byte{}, fmt.Errorf("convert public key to ECDH: %w", err)
	}
	sharedSecret, err := privateKey.ECDH(publicKey)
	if err != nil {
		return [32]byte{}, fmt.Errorf("compute shared secret: %w", err)
	}
	return sha256.Sum256(append(salt, sharedSecret...)), nil
}

func (conn *Conn) expect(packetIDs ...uint32) {
	conn.expectedIDs.Store(packetIDs)
}

func (conn *Conn) closeTransport(cause error) {
	conn.cancelFunc(cause)
	_ = conn.conn.Close()
	_ = conn.close(cause)
}

func (conn *Conn) close(cause error) error {
	var err error
	conn.once.Do(func() {
		err = conn.Flush()
		if conn.cancelFunc != nil {
			conn.cancelFunc(cause)
		}
		if conn.conn != nil {
			_ = conn.conn.Close()
		}
	})
	return err
}

func (conn *Conn) closeErr(op string) error {
	select {
	case <-conn.ctx.Done():
		return conn.wrap(context.Cause(conn.ctx), op)
	default:
		return conn.wrap(net.ErrClosed, op)
	}
}
