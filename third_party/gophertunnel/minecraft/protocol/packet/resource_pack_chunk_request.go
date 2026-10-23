package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ResourcePackChunkRequest struct {
	UUID string

	ChunkIndex int32
}

func (*ResourcePackChunkRequest) ID() uint32 {
	return IDResourcePackChunkRequest
}

func (pk *ResourcePackChunkRequest) Marshal(io protocol.IO) {
	io.String(&pk.UUID)
	io.Int32(&pk.ChunkIndex)
}
