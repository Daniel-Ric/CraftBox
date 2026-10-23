package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type UpdateSubChunkBlocks struct {
	Position protocol.BlockPos

	Blocks []protocol.BlockChangeEntry

	Extra []protocol.BlockChangeEntry
}

func (*UpdateSubChunkBlocks) ID() uint32 {
	return IDUpdateSubChunkBlocks
}

func (pk *UpdateSubChunkBlocks) Marshal(io protocol.IO) {
	io.BlockPos(&pk.Position)
	protocol.Slice(io, &pk.Blocks)
	protocol.Slice(io, &pk.Extra)
}
