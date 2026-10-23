package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type TakeItemActor struct {
	ItemEntityRuntimeID uint64

	TakerEntityRuntimeID uint64
}

func (*TakeItemActor) ID() uint32 {
	return IDTakeItemActor
}

func (pk *TakeItemActor) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.ItemEntityRuntimeID)
	io.ActorRuntimeID(&pk.TakerEntityRuntimeID)
}
