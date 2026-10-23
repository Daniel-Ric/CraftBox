package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type UpdateEquip struct {
	WindowID byte

	WindowType byte

	Size int32

	EntityUniqueID int64

	SerialisedInventoryData []byte
}

func (*UpdateEquip) ID() uint32 {
	return IDUpdateEquip
}

func (pk *UpdateEquip) Marshal(io protocol.IO) {
	io.Uint8(&pk.WindowID)
	io.Uint8(&pk.WindowType)
	io.Varint32(&pk.Size)
	io.ActorUniqueID(&pk.EntityUniqueID)
	io.Bytes(&pk.SerialisedInventoryData)
}
