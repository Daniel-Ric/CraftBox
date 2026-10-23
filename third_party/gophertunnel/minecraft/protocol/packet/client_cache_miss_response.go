package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientCacheMissResponse struct {
	Blobs []protocol.CacheBlob
}

func (pk *ClientCacheMissResponse) ID() uint32 {
	return IDClientCacheMissResponse
}

func (pk *ClientCacheMissResponse) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Blobs)
}
