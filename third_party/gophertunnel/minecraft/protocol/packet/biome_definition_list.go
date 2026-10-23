package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type BiomeDefinitionList struct {
	BiomeDefinitions []protocol.BiomeDefinition

	StringList []string
}

func (*BiomeDefinitionList) ID() uint32 {
	return IDBiomeDefinitionList
}

func (pk *BiomeDefinitionList) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.BiomeDefinitions)
	protocol.FuncSlice(io, &pk.StringList, io.String)
}
