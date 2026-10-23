package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PlayerArmourDamageFlagHelmet = iota
	PlayerArmourDamageFlagChestplate
	PlayerArmourDamageFlagLeggings
	PlayerArmourDamageFlagBoots
	PlayerArmourDamageFlagBody
)

type PlayerArmourDamage struct {
	List []protocol.PlayerArmourDamageEntry
}

func (pk *PlayerArmourDamage) ID() uint32 {
	return IDPlayerArmourDamage
}

func (pk *PlayerArmourDamage) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.List)
}
