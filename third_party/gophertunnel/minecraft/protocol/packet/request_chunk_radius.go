package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type RequestChunkRadius struct {
	ChunkRadius int32

	MaxChunkRadius uint8
}

func (*RequestChunkRadius) ID() uint32 {
	return IDRequestChunkRadius
}

func (pk *RequestChunkRadius) Marshal(io protocol.IO) {
	io.Varint32(&pk.ChunkRadius)
	io.Uint8(&pk.MaxChunkRadius)
}
