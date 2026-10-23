package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AgentAnimation struct {
	Animation byte

	EntityRuntimeID uint64
}

func (*AgentAnimation) ID() uint32 {
	return IDAgentAnimation
}

func (pk *AgentAnimation) Marshal(io protocol.IO) {
	io.Uint8(&pk.Animation)
	io.ActorRuntimeID(&pk.EntityRuntimeID)
}
