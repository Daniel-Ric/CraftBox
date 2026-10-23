package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	SpawnTypePlayer = iota
	SpawnTypeWorld
)

type SetSpawnPosition struct {
	SpawnType int32

	Position protocol.BlockPos

	Dimension int32

	SpawnPosition protocol.BlockPos
}

func (*SetSpawnPosition) ID() uint32 {
	return IDSetSpawnPosition
}

func (pk *SetSpawnPosition) Marshal(io protocol.IO) {
	io.Varint32(&pk.SpawnType)
	io.BlockPos(&pk.Position)
	io.Varint32(&pk.Dimension)
	io.BlockPos(&pk.SpawnPosition)
}
