package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AddBehaviourTree struct {
	BehaviourTree string
}

func (*AddBehaviourTree) ID() uint32 {
	return IDAddBehaviourTree
}

func (pk *AddBehaviourTree) Marshal(io protocol.IO) {
	io.String(&pk.BehaviourTree)
}
