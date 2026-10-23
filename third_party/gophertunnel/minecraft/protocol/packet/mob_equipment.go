package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type MobEquipment struct {
	EntityRuntimeID uint64

	NewItem protocol.ItemInstance

	InventorySlot byte

	HotBarSlot byte

	WindowID byte
}

func (*MobEquipment) ID() uint32 {
	return IDMobEquipment
}

func (pk *MobEquipment) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.ItemInstance(&pk.NewItem)
	io.Uint8(&pk.InventorySlot)
	io.Uint8(&pk.HotBarSlot)
	io.Uint8(&pk.WindowID)
}
