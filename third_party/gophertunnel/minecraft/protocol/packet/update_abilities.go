package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type UpdateAbilities struct {
	AbilityData protocol.AbilityData
}

func (*UpdateAbilities) ID() uint32 {
	return IDUpdateAbilities
}

func (pk *UpdateAbilities) Marshal(io protocol.IO) {
	protocol.Single(io, &pk.AbilityData)
}
