package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type OpenSign struct {
	Position protocol.BlockPos

	FrontSide bool
}

func (*OpenSign) ID() uint32 {
	return IDOpenSign
}

func (pk *OpenSign) Marshal(io protocol.IO) {
	io.BlockPos(&pk.Position)
	io.Bool(&pk.FrontSide)
}
