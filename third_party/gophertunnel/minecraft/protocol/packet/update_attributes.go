package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type UpdateAttributes struct {
	EntityRuntimeID uint64

	Attributes []protocol.Attribute

	Tick uint64
}

func (*UpdateAttributes) ID() uint32 {
	return IDUpdateAttributes
}

func (pk *UpdateAttributes) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	protocol.Slice(io, &pk.Attributes)
	io.Varuint64(&pk.Tick)
}
