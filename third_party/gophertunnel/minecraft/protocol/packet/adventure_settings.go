package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	AdventureFlagWorldImmutable = 1 << iota
	AdventureSettingsFlagsNoPvM
	AdventureSettingsFlagsNoMvP
	AdventureSettingsFlagsUnused
	AdventureSettingsFlagsShowNameTags
	AdventureFlagAutoJump
	AdventureFlagAllowFlight
	AdventureFlagNoClip
	AdventureFlagWorldBuilder
	AdventureFlagFlying
	AdventureFlagMuted
)

const (
	ActionPermissionMine = 1 << iota
	ActionPermissionDoorsAndSwitches
	ActionPermissionOpenContainers
	ActionPermissionAttackPlayers
	ActionPermissionAttackMobs
	ActionPermissionOperator
	ActionPermissionTeleport
	ActionPermissionBuild
	ActionPermissionDefault
)

const (
	PermissionLevelVisitor = iota
	PermissionLevelMember
	PermissionLevelOperator
	PermissionLevelCustom
)

type AdventureSettings struct {
	Flags uint32

	CommandPermissionLevel uint32

	ActionPermissions uint32

	PermissionLevel uint32

	CustomStoredPermissions uint32

	PlayerUniqueID int64
}

func (*AdventureSettings) ID() uint32 {
	return IDAdventureSettings
}

func (pk *AdventureSettings) Marshal(io protocol.IO) {
	io.Varuint32(&pk.Flags)
	io.Varuint32(&pk.CommandPermissionLevel)
	io.Varuint32(&pk.ActionPermissions)
	io.Varuint32(&pk.PermissionLevel)
	io.Varuint32(&pk.CustomStoredPermissions)
	io.ActorUniqueIDInt64(&pk.PlayerUniqueID)
}
