package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ResourcePackTypeAddon = iota + 1
	ResourcePackTypeCached
	ResourcePackTypeCopyProtected
	ResourcePackTypeBehaviour
	ResourcePackTypePersonaPiece
	ResourcePackTypeResources
	ResourcePackTypeSkins
	ResourcePackTypeWorldTemplate
)

type ResourcePackDataInfo struct {
	UUID string

	DataChunkSize uint32

	ChunkCount uint32

	Size uint64

	Hash []byte

	Premium bool

	PackType byte
}

func (*ResourcePackDataInfo) ID() uint32 {
	return IDResourcePackDataInfo
}

func (pk *ResourcePackDataInfo) Marshal(io protocol.IO) {
	io.String(&pk.UUID)
	io.Uint32(&pk.DataChunkSize)
	io.Uint32(&pk.ChunkCount)
	io.Uint64(&pk.Size)
	io.ByteSlice(&pk.Hash)
	io.Bool(&pk.Premium)
	io.Uint8(&pk.PackType)
}
