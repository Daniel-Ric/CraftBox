package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	MobEffectAdd = iota + 1
	MobEffectModify
	MobEffectRemove
)

const (
	EffectSpeed = iota + 1
	EffectSlowness
	EffectHaste
	EffectMiningFatigue
	EffectStrength
	EffectInstantHealth
	EffectInstantDamage
	EffectJumpBoost
	EffectNausea
	EffectRegeneration
	EffectResistance
	EffectFireResistance
	EffectWaterBreathing
	EffectInvisibility
	EffectBlindness
	EffectNightVision
	EffectHunger
	EffectWeakness
	EffectPoison
	EffectWither
	EffectHealthBoost
	EffectAbsorption
	EffectSaturation
	EffectLevitation
	EffectFatalPoison
	EffectConduitPower
	EffectSlowFalling
)

type MobEffect struct {
	EntityRuntimeID uint64

	Operation byte

	EffectType int32

	Amplifier int32

	Particles bool

	Duration int32

	Tick uint64

	Ambient bool
}

func (*MobEffect) ID() uint32 {
	return IDMobEffect
}

func (pk *MobEffect) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.Uint8(&pk.Operation)
	io.Varint32(&pk.EffectType)
	io.Varint32(&pk.Amplifier)
	io.Bool(&pk.Particles)
	io.Varint32(&pk.Duration)
	io.Varuint64(&pk.Tick)
	io.Bool(&pk.Ambient)
}
