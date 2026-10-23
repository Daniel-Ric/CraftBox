package minecraft

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

type resourcePackQueue struct {
	packs           []*resource.Pack
	packsToDownload map[string]*resource.Pack
	currentPack     *resource.Pack
	currentOffset   uint64

	packAmount       int
	downloadingPacks map[string]downloadingPack
	awaitingPacks    map[string]*downloadingPack
	chunkSize        uint32
}

type downloadingPack struct {
	buf           *bytes.Buffer
	chunkSize     uint32
	size          uint64
	expectedIndex uint32
	newFrag       chan []byte
	contentKey    string
	cacheKey      ResourcePackCacheKey
}

func (queue *resourcePackQueue) Request(packs []string) error {
	queue.packsToDownload = make(map[string]*resource.Pack)
	for _, requestedPackID := range packs {
		uuid, version, hasVersion := strings.Cut(requestedPackID, "_")
		found := false
		for _, pack := range queue.packs {
			if uuid == pack.UUID().String() && (!hasVersion || version == "" || version == pack.Version()) {
				queue.packsToDownload[pack.UUID().String()] = pack
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("resource pack (UUID=%v) not found", requestedPackID)
		}
	}
	return nil
}

func (queue *resourcePackQueue) NextPack() (pk *packet.ResourcePackDataInfo, ok bool) {
	for index, pack := range queue.packsToDownload {
		delete(queue.packsToDownload, index)

		queue.currentPack = pack
		queue.currentOffset = 0
		checksum := pack.Checksum()

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
		return &packet.ResourcePackDataInfo{
			UUID:          pack.UUID().String() + "_" + pack.Version(),
			DataChunkSize: queue.chunkSize,
			ChunkCount:    uint32(pack.DataChunkCount(int(queue.chunkSize))),
			Size:          uint64(pack.Len()),
			Hash:          checksum[:],
			PackType:      packType,
		}, true
	}
	return nil, false
}

func (queue *resourcePackQueue) AllDownloaded() bool {
	return len(queue.packsToDownload) == 0
}
