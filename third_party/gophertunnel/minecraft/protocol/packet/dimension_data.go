package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type DimensionData struct {
	Definitions []protocol.DimensionDefinition
}

func (*DimensionData) ID() uint32 {
	return IDDimensionData
}

func (pk *DimensionData) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Definitions)
}
