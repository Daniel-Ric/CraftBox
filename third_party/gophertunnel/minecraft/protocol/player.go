package protocol

import (
	"image/color"

	"github.com/google/uuid"
)

const (
	PlayerListActionAdd = iota
	PlayerListActionRemove
)

const (
	PlayerActionStartBreak = iota
	PlayerActionAbortBreak
	PlayerActionStopBreak
	PlayerActionGetUpdatedBlock
	PlayerActionDropItem
	PlayerActionStartSleeping
	PlayerActionStopSleeping
	PlayerActionRespawn
	PlayerActionJump
	PlayerActionStartSprint
	PlayerActionStopSprint
	PlayerActionStartSneak
	PlayerActionStopSneak
	PlayerActionCreativePlayerDestroyBlock
	PlayerActionDimensionChangeDone
	PlayerActionStartGlide
	PlayerActionStopGlide
	PlayerActionBuildDenied
	PlayerActionCrackBreak
	PlayerActionChangeSkin
	PlayerActionSetEnchantmentSeed
	PlayerActionStartSwimming
	PlayerActionStopSwimming
	PlayerActionStartSpinAttack
	PlayerActionStopSpinAttack
	PlayerActionInteractWithBlock
	PlayerActionPredictDestroyBlock
	PlayerActionContinueDestroyBlock
	PlayerActionStartItemUseOn
	PlayerActionStopItemUseOn
	PlayerActionHandledTeleport
	PlayerActionMissedSwing
	PlayerActionStartCrawling
	PlayerActionStopCrawling
	PlayerActionStartFlying
	PlayerActionStopFlying
	PlayerActionReceivedServerData
	PlayerActionStartUsingItem
	PlayerActionInternalUpdate
	PlayerActionCount
)

type PlayerListEntry struct {
	ActionType byte

	UUID uuid.UUID

	EntityUniqueID int64

	Username string

	XUID string

	PlatformChatID string

	BuildPlatform int32

	Skin Skin

	Teacher bool

	Host bool

	SubClient bool

	PlayerColour color.RGBA
}

func (x *PlayerListEntry) Marshal(r IO) {
	playerListAction(r, &x.ActionType)
	r.UUID(&x.UUID)
	if x.ActionType == PlayerListActionRemove {
		return
	}

	r.ActorUniqueID(&x.EntityUniqueID)
	r.String(&x.Username)
	r.String(&x.XUID)
	r.String(&x.PlatformChatID)
	r.Int32(&x.BuildPlatform)
	Single(r, &x.Skin)
	r.Bool(&x.Teacher)
	r.Bool(&x.Host)
	r.Bool(&x.SubClient)
	r.BEARGB(&x.PlayerColour)
}

func playerListAction(r IO, action *byte) {
	variant := uint32(0)
	if *action == PlayerListActionAdd {
		variant = 1
	}
	r.Varuint32(&variant)

	legacyAction := *action
	r.Uint8(&legacyAction)
	*action = PlayerListActionRemove
	if variant == 1 {
		*action = PlayerListActionAdd
	} else if variant != 0 {
		r.UnknownEnumOption(variant, "player list entry variant")
	}
}

type PlayerMovementSettings struct {
	RewindHistorySize int32

	ServerAuthoritativeBlockBreaking bool
}

func PlayerMoveSettings(r IO, x *PlayerMovementSettings) {
	r.Varint32(&x.RewindHistorySize)
	r.Bool(&x.ServerAuthoritativeBlockBreaking)
}

type PlayerBlockAction struct {
	Action int32

	BlockPos BlockPos

	Face int32
}

func (x *PlayerBlockAction) Marshal(r IO) {
	r.Varint32(&x.Action)
	r.BlockPos(&x.BlockPos)
	r.Varint32(&x.Face)
}

type PlayerArmourDamageEntry struct {
	ArmourSlot int32

	Damage int16
}

func (x *PlayerArmourDamageEntry) Marshal(r IO) {
	r.Varint32(&x.ArmourSlot)
	r.Int16(&x.Damage)
}

type TeleportData struct {
	TeleportCause int32

	TeleportSourceEntityType int32
}

func (x *TeleportData) Marshal(r IO) {
	r.Int32(&x.TeleportCause)
	r.Int32(&x.TeleportSourceEntityType)
}
