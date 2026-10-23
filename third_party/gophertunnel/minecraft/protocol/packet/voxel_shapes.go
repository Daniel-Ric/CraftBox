package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type VoxelShapes struct {
	Shapes []protocol.VoxelShape

	NameMap []protocol.VoxelShapeNameEntry

	CustomShapeCount uint16
}

func (*VoxelShapes) ID() uint32 {
	return IDVoxelShapes
}

func (pk *VoxelShapes) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Shapes)
	protocol.Slice(io, &pk.NameMap)
	io.Uint16(&pk.CustomShapeCount)
}
