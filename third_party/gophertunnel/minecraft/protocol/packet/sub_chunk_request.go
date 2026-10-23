package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SubChunkRequest struct {
	Dimension int32

	Offsets []protocol.SubChunkOffset

	Position protocol.SubChunkPos
}

func (*SubChunkRequest) ID() uint32 {
	return IDSubChunkRequest
}

func (pk *SubChunkRequest) Marshal(io protocol.IO) {
	io.Varint32(&pk.Dimension)
	protocol.Slice(io, &pk.Offsets)
	io.SubChunkPos(&pk.Position)
}
