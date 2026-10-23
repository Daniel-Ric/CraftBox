package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type RemoveObjective struct {
	ObjectiveName string
}

func (*RemoveObjective) ID() uint32 {
	return IDRemoveObjective
}

func (pk *RemoveObjective) Marshal(io protocol.IO) {
	io.String(&pk.ObjectiveName)
}
