package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type LevelChunk struct {
	Position protocol.ChunkPos

	Dimension     int32
	SubChunkCount uint32

	SubChunkLimit protocol.Optional[int32]

	CacheEnabled bool

	BlobHashes []uint64

	RawPayload []byte
}

func (*LevelChunk) ID() uint32 {
	return IDLevelChunk
}

func (pk *LevelChunk) Marshal(io protocol.IO) {
	io.ChunkPos(&pk.Position)
	io.Varint32(&pk.Dimension)
	io.Varuint32(&pk.SubChunkCount)
	if pk.SubChunkCount > 64 {
		io.InvalidValue(pk.SubChunkCount, "level chunk sub-chunk count", "must not exceed 64")
	}
	protocol.OptionalFunc(io, &pk.SubChunkLimit, io.Varint32)
	io.Bool(&pk.CacheEnabled)
	protocol.FuncSlice(io, &pk.BlobHashes, io.Uint64)
	io.ByteSlice(&pk.RawPayload)
}
