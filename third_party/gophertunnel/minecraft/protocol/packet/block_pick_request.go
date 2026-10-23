package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type BlockPickRequest struct {
	Position protocol.BlockPos

	AddBlockNBT bool

	HotBarSlot byte
}

func (*BlockPickRequest) ID() uint32 {
	return IDBlockPickRequest
}

func (pk *BlockPickRequest) Marshal(io protocol.IO) {
	io.BlockPos(&pk.Position)
	io.Bool(&pk.AddBlockNBT)
	io.Uint8(&pk.HotBarSlot)
}
