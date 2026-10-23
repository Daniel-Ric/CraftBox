package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetHealth struct {
	Health int32
}

func (*SetHealth) ID() uint32 {
	return IDSetHealth
}

func (pk *SetHealth) Marshal(io protocol.IO) {
	io.Varint32(&pk.Health)
}
