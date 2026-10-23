package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type InventoryContent struct {
	WindowID uint32

	Content []protocol.ItemInstance

	Container protocol.FullContainerName

	StorageItem protocol.ItemInstance
}

func (*InventoryContent) ID() uint32 {
	return IDInventoryContent
}

func (pk *InventoryContent) Marshal(io protocol.IO) {
	io.Varuint32(&pk.WindowID)
	protocol.FuncSlice(io, &pk.Content, io.ItemInstance)
	protocol.Single(io, &pk.Container)
	io.ItemInstance(&pk.StorageItem)
}
