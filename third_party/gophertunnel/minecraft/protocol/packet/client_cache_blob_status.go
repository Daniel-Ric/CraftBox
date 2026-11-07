package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientCacheBlobStatus struct {
	MissHashes []uint64

	HitHashes []uint64
}

func (pk *ClientCacheBlobStatus) ID() uint32 {
	return IDClientCacheBlobStatus
}

func (pk *ClientCacheBlobStatus) Marshal(io protocol.IO) {
	clientCacheHashes(io, &pk.MissHashes)
	clientCacheHashes(io, &pk.HitHashes)
}

// Clients batch cache acknowledgements across many chunks. Each MISS/HIT list
// supports 4095 hashes, unlike protocol.FuncSlice's generic 1024-entry limit.
// Keep the Cereal wire order: miss count+hashes, followed by hit count+hashes.
func clientCacheHashes(io protocol.IO, hashes *[]uint64) {
	count := uint32(len(*hashes))
	io.Varuint32(&count)
	if reader, ok := io.(interface{ SliceLength(uint32, uint32) }); ok {
		reader.SliceLength(count, 4095)
		*hashes = make([]uint64, count)
	}
	for i := uint32(0); i < count; i++ {
		io.Uint64(&(*hashes)[i])
	}
}
