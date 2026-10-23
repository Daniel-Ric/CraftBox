package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AddVolumeEntity struct {
	EntityRuntimeID uint32

	EntityMetadata map[string]any

	EncodingIdentifier string

	InstanceIdentifier string

	Bounds [2]protocol.BlockPos

	Dimension int32

	EngineVersion string
}

func (*AddVolumeEntity) ID() uint32 {
	return IDAddVolumeEntity
}

func (pk *AddVolumeEntity) Marshal(io protocol.IO) {
	io.ActorRuntimeIDVaruint32(&pk.EntityRuntimeID)
	io.NBT(&pk.EntityMetadata, nbt.NetworkLittleEndian)
	io.String(&pk.EncodingIdentifier)
	io.String(&pk.InstanceIdentifier)
	io.BlockPos(&pk.Bounds[0])
	io.BlockPos(&pk.Bounds[1])
	io.Varint32(&pk.Dimension)
	io.String(&pk.EngineVersion)
}
