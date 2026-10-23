package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	StructureBlockData = iota
	StructureBlockSave
	StructureBlockLoad
	StructureBlockCorner
	StructureBlockInvalid
	StructureBlockExport
)

const (
	StructureRedstoneSaveModeMemory = iota
	StructureRedstoneSaveModeDisk
)

type StructureBlockUpdate struct {
	Position protocol.BlockPos

	StructureName string

	FilteredStructureName string

	DataField string

	IncludePlayers bool

	ShowBoundingBox bool

	StructureBlockType int32

	Settings protocol.StructureSettings

	RedstoneSaveMode uint8

	ShouldTrigger bool

	Waterlogged bool
}

func (*StructureBlockUpdate) ID() uint32 {
	return IDStructureBlockUpdate
}

func (pk *StructureBlockUpdate) Marshal(io protocol.IO) {
	io.BlockPos(&pk.Position)
	io.String(&pk.StructureName)
	io.String(&pk.FilteredStructureName)
	io.String(&pk.DataField)
	io.Bool(&pk.IncludePlayers)
	io.Bool(&pk.ShowBoundingBox)
	io.Varint32(&pk.StructureBlockType)
	protocol.Single(io, &pk.Settings)
	io.Uint8(&pk.RedstoneSaveMode)
	io.Bool(&pk.ShouldTrigger)
	io.Bool(&pk.Waterlogged)
}
