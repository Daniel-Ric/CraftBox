package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	InventoryTransactionTypeNormal = iota
	InventoryTransactionTypeMismatch
	InventoryTransactionTypeUseItem
	InventoryTransactionTypeUseItemOnEntity
	InventoryTransactionTypeReleaseItem
)

type InventoryTransaction struct {
	LegacyRequestID int32

	LegacySetItemSlots []protocol.LegacySetItemSlot

	Actions []protocol.InventoryAction

	TransactionData protocol.InventoryTransactionData
}

func (*InventoryTransaction) ID() uint32 {
	return IDInventoryTransaction
}

func (pk *InventoryTransaction) Marshal(io protocol.IO) {
	io.Varint32(&pk.LegacyRequestID)
	hasLegacy := pk.LegacyRequestID < -1 && (pk.LegacyRequestID&1) == 0
	io.Bool(&hasLegacy)
	if hasLegacy {
		protocol.Slice(io, &pk.LegacySetItemSlots)
	}
	io.TransactionDataType(&pk.TransactionData)
	protocol.Slice(io, &pk.Actions)
	pk.TransactionData.Marshal(io)
}
