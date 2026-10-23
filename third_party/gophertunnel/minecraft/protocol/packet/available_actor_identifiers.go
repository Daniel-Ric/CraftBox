package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AvailableActorIdentifiers struct {
	SerialisedEntityIdentifiers []byte
}

func (*AvailableActorIdentifiers) ID() uint32 {
	return IDAvailableActorIdentifiers
}

func (pk *AvailableActorIdentifiers) Marshal(io protocol.IO) {
	io.Bytes(&pk.SerialisedEntityIdentifiers)
}
