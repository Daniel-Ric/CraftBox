package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetActorLink struct {
	EntityLink protocol.EntityLink
}

func (*SetActorLink) ID() uint32 {
	return IDSetActorLink
}

func (pk *SetActorLink) Marshal(io protocol.IO) {
	protocol.Single(io, &pk.EntityLink)
}
