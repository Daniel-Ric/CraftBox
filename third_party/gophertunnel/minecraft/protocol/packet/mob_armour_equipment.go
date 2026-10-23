package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type MobArmourEquipment struct {
	EntityRuntimeID uint64

	Helmet protocol.ItemInstance

	Chestplate protocol.ItemInstance

	Leggings protocol.ItemInstance

	Boots protocol.ItemInstance

	Body protocol.ItemInstance
}

func (*MobArmourEquipment) ID() uint32 {
	return IDMobArmourEquipment
}

func (pk *MobArmourEquipment) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.ItemInstance(&pk.Helmet)
	io.ItemInstance(&pk.Chestplate)
	io.ItemInstance(&pk.Leggings)
	io.ItemInstance(&pk.Boots)
	io.ItemInstance(&pk.Body)
}
