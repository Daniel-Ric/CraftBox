package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ItemRegistry struct {
	Items []protocol.ItemEntry
}

func (*ItemRegistry) ID() uint32 {
	return IDItemRegistry
}

func (pk *ItemRegistry) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Items)
}
