package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type RemoveActor struct {
	EntityUniqueID int64
}

func (*RemoveActor) ID() uint32 {
	return IDRemoveActor
}

func (pk *RemoveActor) Marshal(io protocol.IO) {
	io.ActorUniqueID(&pk.EntityUniqueID)
}
