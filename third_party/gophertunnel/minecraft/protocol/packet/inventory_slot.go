package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type InventorySlot struct {
	WindowID uint32

	Slot uint32

	Container protocol.Optional[protocol.FullContainerName]

	StorageItem protocol.Optional[protocol.ItemInstance]

	NewItem protocol.ItemInstance
}

func (*InventorySlot) ID() uint32 {
	return IDInventorySlot
}

func (pk *InventorySlot) Marshal(io protocol.IO) {
	io.Varuint32(&pk.WindowID)
	io.Varuint32(&pk.Slot)
	protocol.OptionalMarshaler(io, &pk.Container)
	protocol.OptionalFunc(io, &pk.StorageItem, io.ItemInstance)
	io.ItemInstance(&pk.NewItem)
}
