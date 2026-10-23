package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SubChunk struct {
	CacheEnabled bool

	Dimension int32

	Position protocol.SubChunkPos

	SubChunkEntries []protocol.SubChunkEntry
}

func (*SubChunk) ID() uint32 {
	return IDSubChunk
}

func (pk *SubChunk) Marshal(io protocol.IO) {
	io.Bool(&pk.CacheEnabled)
	io.Varint32(&pk.Dimension)
	io.SubChunkPos(&pk.Position)
	protocol.Slice(io, &pk.SubChunkEntries)
}
