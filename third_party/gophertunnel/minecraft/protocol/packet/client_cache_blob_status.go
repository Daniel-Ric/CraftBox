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
	protocol.FuncSlice(io, &pk.MissHashes, io.Uint64)
	protocol.FuncSlice(io, &pk.HitHashes, io.Uint64)
}
