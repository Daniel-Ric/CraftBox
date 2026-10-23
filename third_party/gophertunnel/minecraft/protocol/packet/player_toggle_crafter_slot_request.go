package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PlayerToggleCrafterSlotRequest struct {
	PosX int32

	PosY int32

	PosZ int32

	Slot byte

	Disabled bool
}

func (*PlayerToggleCrafterSlotRequest) ID() uint32 {
	return IDPlayerToggleCrafterSlotRequest
}

func (pk *PlayerToggleCrafterSlotRequest) Marshal(io protocol.IO) {
	io.Int32(&pk.PosX)
	io.Int32(&pk.PosY)
	io.Int32(&pk.PosZ)
	io.Uint8(&pk.Slot)
	io.Bool(&pk.Disabled)
}
