package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	AbilityBuild = iota
	AbilityMine
	AbilityDoorsAndSwitches
	AbilityOpenContainers
	AbilityAttackPlayers
	AbilityAttackMobs
	AbilityOperatorCommands
	AbilityTeleport
	AbilityInvulnerable
	AbilityFlying
	AbilityMayFly
	AbilityInstantBuild
	AbilityLightning
	AbilityFlySpeed
	AbilityWalkSpeed
	AbilityMuted
	AbilityWorldBuilder
	AbilityNoClip
	AbilityCount
)

type RequestAbility struct {
	Ability int32

	Value any
}

func (*RequestAbility) ID() uint32 {
	return IDRequestAbility
}

func (pk *RequestAbility) Marshal(io protocol.IO) {
	io.Varint32(&pk.Ability)
	io.AbilityValue(&pk.Value)
}
