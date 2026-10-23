package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type NetworkChunkPublisherUpdate struct {
	Position protocol.BlockPos

	Radius uint32

	SavedChunks []protocol.ChunkPos
}

func (*NetworkChunkPublisherUpdate) ID() uint32 {
	return IDNetworkChunkPublisherUpdate
}

func (pk *NetworkChunkPublisherUpdate) Marshal(io protocol.IO) {
	io.BlockPos(&pk.Position)
	io.Varuint32(&pk.Radius)
	count := uint32(len(pk.SavedChunks))
	io.Uint32(&count)
	if count > 9216 {
		io.InvalidValue(count, "savedChunks", "saved chunks exceeds maximum length of 9216 chunk positions")
	}
	protocol.FuncSliceOfLen(io, count, &pk.SavedChunks, io.ChunkPos)
}
