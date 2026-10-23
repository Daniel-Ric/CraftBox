package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ActorPickRequest struct {
	EntityUniqueID int64

	HotBarSlot byte

	WithData bool
}

func (*ActorPickRequest) ID() uint32 {
	return IDActorPickRequest
}

func (pk *ActorPickRequest) Marshal(io protocol.IO) {
	io.ActorUniqueIDInt64(&pk.EntityUniqueID)
	io.Uint8(&pk.HotBarSlot)
	io.Bool(&pk.WithData)
}
