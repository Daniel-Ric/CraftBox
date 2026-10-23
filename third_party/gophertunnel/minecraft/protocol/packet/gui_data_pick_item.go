package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type GUIDataPickItem struct {
	ItemName string

	ItemEffects string

	HotBarSlot int32
}

func (*GUIDataPickItem) ID() uint32 {
	return IDGUIDataPickItem
}

func (pk *GUIDataPickItem) Marshal(io protocol.IO) {
	io.String(&pk.ItemName)
	io.String(&pk.ItemEffects)
	io.Int32(&pk.HotBarSlot)
}
