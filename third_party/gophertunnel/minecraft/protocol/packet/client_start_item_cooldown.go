package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientStartItemCooldown struct {
	Category string

	Duration int32
}

func (*ClientStartItemCooldown) ID() uint32 {
	return IDClientStartItemCooldown
}

func (pk *ClientStartItemCooldown) Marshal(io protocol.IO) {
	io.String(&pk.Category)
	io.Varint32(&pk.Duration)
}
