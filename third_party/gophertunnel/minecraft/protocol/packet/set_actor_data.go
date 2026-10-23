package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetActorData struct {
	EntityRuntimeID uint64

	EntityMetadata protocol.EntityMetadata

	EntityProperties protocol.EntityProperties

	Tick uint64
}

func (*SetActorData) ID() uint32 {
	return IDSetActorData
}

func (pk *SetActorData) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.EntityMetadata(&pk.EntityMetadata)
	protocol.Single(io, &pk.EntityProperties)
	io.Varuint64(&pk.Tick)
}
