package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AnvilDamage struct {
	AnvilPosition protocol.BlockPos
}

func (*AnvilDamage) ID() uint32 {
	return IDAnvilDamage
}

func (pk *AnvilDamage) Marshal(io protocol.IO) {
	io.BlockPos(&pk.AnvilPosition)
}
