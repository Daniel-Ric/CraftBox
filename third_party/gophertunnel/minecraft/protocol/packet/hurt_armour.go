package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type HurtArmour struct {
	Cause int32

	Damage int32

	ArmourSlots uint64
}

func (*HurtArmour) ID() uint32 {
	return IDHurtArmour
}

func (pk *HurtArmour) Marshal(io protocol.IO) {
	io.Varint32(&pk.Cause)
	io.Varint32(&pk.Damage)
	io.Varuint64(&pk.ArmourSlots)
}
