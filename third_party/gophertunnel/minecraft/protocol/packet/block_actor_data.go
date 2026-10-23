package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type BlockActorData struct {
	Position protocol.BlockPos

	NBTData map[string]any
}

func (*BlockActorData) ID() uint32 {
	return IDBlockActorData
}

func (pk *BlockActorData) Marshal(io protocol.IO) {
	io.BlockPos(&pk.Position)
	io.NBT(&pk.NBTData, nbt.NetworkLittleEndian)
}
