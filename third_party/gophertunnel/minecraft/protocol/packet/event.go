package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type Event struct {
	EntityRuntimeID int64

	UsePlayerID bool

	Event protocol.Event
}

func (*Event) ID() uint32 {
	return IDEvent
}

func (pk *Event) Marshal(io protocol.IO) {
	io.ActorRuntimeIDVarint64(&pk.EntityRuntimeID)
	io.EventType(&pk.Event)
	io.Bool(&pk.UsePlayerID)
	io.EventOrdinal(&pk.Event)
	pk.Event.Marshal(io)
}
