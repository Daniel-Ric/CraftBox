package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetLastHurtBy struct {
	EntityType int32
}

func (*SetLastHurtBy) ID() uint32 {
	return IDSetLastHurtBy
}

func (pk *SetLastHurtBy) Marshal(io protocol.IO) {
	io.Varint32(&pk.EntityType)
}
