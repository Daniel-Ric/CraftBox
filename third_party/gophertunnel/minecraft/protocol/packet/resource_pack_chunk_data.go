package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ResourcePackChunkData struct {
	UUID string

	ChunkIndex uint32

	DataOffset uint64

	Data []byte
}

func (*ResourcePackChunkData) ID() uint32 {
	return IDResourcePackChunkData
}

func (pk *ResourcePackChunkData) Marshal(io protocol.IO) {
	io.String(&pk.UUID)
	io.Uint32(&pk.ChunkIndex)
	io.Uint64(&pk.DataOffset)
	io.ByteSlice(&pk.Data)
}
