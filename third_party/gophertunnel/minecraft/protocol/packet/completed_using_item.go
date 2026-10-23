package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	UseItemEquipArmour = iota
	UseItemEat
	UseItemAttack
	UseItemConsume
	UseItemThrow
	UseItemShoot
	UseItemPlace
	UseItemFillBottle
	UseItemFillBucket
	UseItemPourBucket
	UseItemUseTool
	UseItemInteract
	UseItemRetrieved
	UseItemDyed
	UseItemTraded
	UseItemBrushingCompleted
	UseItemOpenedVault
)

type CompletedUsingItem struct {
	UsedItemID int16

	UseMethod int32
}

func (*CompletedUsingItem) ID() uint32 {
	return IDCompletedUsingItem
}

func (pk *CompletedUsingItem) Marshal(io protocol.IO) {
	io.Int16(&pk.UsedItemID)
	io.Int32(&pk.UseMethod)
}
