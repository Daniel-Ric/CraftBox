package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PlayerHotBar struct {
	SelectedHotBarSlot uint32

	WindowID byte

	SelectHotBarSlot bool
}

func (*PlayerHotBar) ID() uint32 {
	return IDPlayerHotBar
}

func (pk *PlayerHotBar) Marshal(io protocol.IO) {
	io.Varuint32(&pk.SelectedHotBarSlot)
	io.Uint8(&pk.WindowID)
	io.Bool(&pk.SelectHotBarSlot)
}
