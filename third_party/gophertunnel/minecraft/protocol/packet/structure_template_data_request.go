package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	StructureTemplateRequestExportFromSave = iota + 1
	StructureTemplateRequestExportFromLoad
	StructureTemplateRequestQuerySavedStructure
)

type StructureTemplateDataRequest struct {
	StructureName string

	Position protocol.BlockPos

	Settings protocol.StructureSettings

	RequestType byte
}

func (pk *StructureTemplateDataRequest) ID() uint32 {
	return IDStructureTemplateDataRequest
}

func (pk *StructureTemplateDataRequest) Marshal(io protocol.IO) {
	io.String(&pk.StructureName)
	io.BlockPos(&pk.Position)
	protocol.Single(io, &pk.Settings)
	io.Uint8(&pk.RequestType)
}
