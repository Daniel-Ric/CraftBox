package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientCheatAbility struct {
	AbilityData protocol.AbilityData
}

func (*ClientCheatAbility) ID() uint32 {
	return IDClientCheatAbility
}

func (pk *ClientCheatAbility) Marshal(io protocol.IO) {
	protocol.Single(io, &pk.AbilityData)
}
