package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PlayerEnchantOptions struct {
	Options []protocol.EnchantmentOption
}

func (*PlayerEnchantOptions) ID() uint32 {
	return IDPlayerEnchantOptions
}

func (pk *PlayerEnchantOptions) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Options)
}
