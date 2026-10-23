package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ChunkRadiusUpdated struct {
	ChunkRadius int32
}

func (*ChunkRadiusUpdated) ID() uint32 {
	return IDChunkRadiusUpdated
}

func (pk *ChunkRadiusUpdated) Marshal(io protocol.IO) {
	io.Varint32(&pk.ChunkRadius)
}
