package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type CurrentStructureFeature struct {
	CurrentFeature string
}

func (*CurrentStructureFeature) ID() uint32 {
	return IDCurrentStructureFeature
}

func (pk *CurrentStructureFeature) Marshal(io protocol.IO) {
	io.String(&pk.CurrentFeature)
}
