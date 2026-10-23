package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	MovementEffectTypeGlideBoost = iota
	MovementEffectTypeDolphinBoost
	MovementEffectTypeGeyserBoost
)

type MovementEffect struct {
	EntityRuntimeID uint64

	Type int32

	Duration int32

	Tick uint64
}

func (*MovementEffect) ID() uint32 {
	return IDMovementEffect
}

func (pk *MovementEffect) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.Varint32(&pk.Type)
	io.Varint32(&pk.Duration)
	io.Varuint64(&pk.Tick)
}
