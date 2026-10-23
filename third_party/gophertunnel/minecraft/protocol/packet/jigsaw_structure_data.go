package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type JigsawStructureData struct {
	StructureData map[string]any
}

func (*JigsawStructureData) ID() uint32 {
	return IDJigsawStructureData
}

func (pk *JigsawStructureData) Marshal(io protocol.IO) {
	io.NBT(&pk.StructureData, nbt.NetworkLittleEndian)
}
