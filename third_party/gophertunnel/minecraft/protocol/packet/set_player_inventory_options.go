package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	InventoryLayoutNone = iota
	InventoryLayoutInventoryOnly
	InventoryLayoutDefault
	InventoryLayoutRecipeBookOnly
)

const (
	InventoryLeftTabNone = iota
	InventoryLeftTabConstruction
	InventoryLeftTabEquipment
	InventoryLeftTabItems
	InventoryLeftTabNature
	InventoryLeftTabSearch
	InventoryLeftTabSurvival
)

const (
	InventoryRightTabNone = iota
	InventoryRightTabFullScreen
	InventoryRightTabCrafting
	InventoryRightTabArmour
)

type SetPlayerInventoryOptions struct {
	LeftInventoryTab int32

	RightInventoryTab int32

	Filtering bool

	InventoryLayout int32

	CraftingLayout int32
}

func (*SetPlayerInventoryOptions) ID() uint32 {
	return IDSetPlayerInventoryOptions
}

func (pk *SetPlayerInventoryOptions) Marshal(io protocol.IO) {
	io.Varint32(&pk.LeftInventoryTab)
	io.Varint32(&pk.RightInventoryTab)
	io.Bool(&pk.Filtering)
	io.Varint32(&pk.InventoryLayout)
	io.Varint32(&pk.CraftingLayout)
}
