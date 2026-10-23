package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetLocalPlayerAsInitialised struct {
	EntityRuntimeID uint64
}

func (*SetLocalPlayerAsInitialised) ID() uint32 {
	return IDSetLocalPlayerAsInitialised
}

func (pk *SetLocalPlayerAsInitialised) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
}
