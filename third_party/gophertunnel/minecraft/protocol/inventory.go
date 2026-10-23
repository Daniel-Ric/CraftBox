package protocol

import (
	"github.com/go-gl/mathgl/mgl32"
)

const (
	InventoryActionSourceContainer = 0
	InventoryActionSourceWorld     = 2
	InventoryActionSourceCreative  = 3
	InventoryActionSourceTODO      = 99999
)

const (
	WindowIDInventory = 0
	WindowIDOffHand   = 119
	WindowIDArmour    = 120
	WindowIDUI        = 124
)

type InventoryAction struct {
	SourceType uint32

	WindowID Optional[int8]

	SourceFlags Optional[uint32]

	InventorySlot uint32

	OldItem ItemInstance

	NewItem ItemInstance
}

func (x *InventoryAction) Marshal(r IO) {
	r.Varuint32(&x.SourceType)
	OptionalFunc(r, &x.WindowID, r.Int8)
	OptionalFunc(r, &x.SourceFlags, r.Varuint32)
	r.Varuint32(&x.InventorySlot)
	r.ItemInstance(&x.OldItem)
	r.ItemInstance(&x.NewItem)
}

const (
	InventoryTransactionTypeNormal = iota
	InventoryTransactionTypeMismatch
	InventoryTransactionTypeUseItem
	InventoryTransactionTypeUseItemOnEntity
	InventoryTransactionTypeReleaseItem
)

type InventoryTransactionData interface {
	Marshal(r IO)
}

func lookupTransactionData(id uint32, x *InventoryTransactionData) bool {
	switch id {
	case InventoryTransactionTypeNormal:
		*x = &NormalTransactionData{}
	case InventoryTransactionTypeMismatch:
		*x = &MismatchTransactionData{}
	case InventoryTransactionTypeUseItem:
		*x = &UseItemTransactionData{}
	case InventoryTransactionTypeUseItemOnEntity:
		*x = &UseItemOnEntityTransactionData{}
	case InventoryTransactionTypeReleaseItem:
		*x = &ReleaseItemTransactionData{}
	default:
		return false
	}
	return true
}

func lookupTransactionDataType(x InventoryTransactionData, id *uint32) bool {
	switch x.(type) {
	case *NormalTransactionData:
		*id = InventoryTransactionTypeNormal
	case *MismatchTransactionData:
		*id = InventoryTransactionTypeMismatch
	case *UseItemTransactionData:
		*id = InventoryTransactionTypeUseItem
	case *UseItemOnEntityTransactionData:
		*id = InventoryTransactionTypeUseItemOnEntity
	case *ReleaseItemTransactionData:
		*id = InventoryTransactionTypeReleaseItem
	default:
		return false
	}
	return true
}

type NormalTransactionData struct{}

type MismatchTransactionData struct{}

const (
	UseItemActionClickBlock = iota
	UseItemActionClickAir
	UseItemActionBreakBlock
	UseItemActionUseAsAttack
)

const (
	TriggerTypeUnknown = iota
	TriggerTypePlayerInput
	TriggerTypeSimulationTick
)

const (
	ClientPredictionFailure = iota
	ClientPredictionSuccess
)

const (
	ClientCooldownStateOff = iota
	ClientCooldownStateOn
)

const (
	HandSlotMainHand = iota
	HandSlotOffHand
)

type UseItemTransactionData struct {
	LegacyRequestID int32

	LegacySetItemSlots Optional[[]LegacySetItemSlot]

	Actions []InventoryAction

	ActionType uint32

	TriggerType uint32

	BlockPosition BlockPos

	BlockFace int32

	HotBarSlot int32

	Hand byte

	HeldItem ItemInstance

	Position mgl32.Vec3

	ClickedPosition mgl32.Vec3

	BlockRuntimeID uint32

	ClientPrediction uint8

	ClientCooldownState byte
}

const (
	UseItemOnEntityActionInteract = iota
	UseItemOnEntityActionAttack
)

type UseItemOnEntityTransactionData struct {
	TargetEntityRuntimeID uint64

	ActionType int32

	HotBarSlot int32

	HeldItem ItemInstance

	Position mgl32.Vec3

	ClickedPosition mgl32.Vec3
}

const (
	ReleaseItemActionRelease = iota
	ReleaseItemActionConsume
)

type ReleaseItemTransactionData struct {
	ActionType int32

	HotBarSlot int32

	HeldItem ItemInstance

	HeadPosition mgl32.Vec3
}

func (data *UseItemTransactionData) Marshal(r IO) {
	IntegerFunc(&data.ActionType, r.Varint32)
	IntegerFunc(&data.TriggerType, r.Uint8)
	r.BlockPos(&data.BlockPosition)
	IntegerFunc(&data.BlockFace, r.Uint8)
	r.Varint32(&data.HotBarSlot)
	r.Uint8(&data.Hand)
	r.ItemInstance(&data.HeldItem)
	r.Vec3(&data.Position)
	r.Vec3(&data.ClickedPosition)
	r.Varuint32(&data.BlockRuntimeID)
	r.Uint8(&data.ClientPrediction)
	r.Uint8(&data.ClientCooldownState)
}

func (data *UseItemOnEntityTransactionData) Marshal(r IO) {
	r.ActorRuntimeID(&data.TargetEntityRuntimeID)
	r.Varint32(&data.ActionType)
	r.Varint32(&data.HotBarSlot)
	r.ItemInstance(&data.HeldItem)
	r.Vec3(&data.Position)
	r.Vec3(&data.ClickedPosition)
}

func (data *ReleaseItemTransactionData) Marshal(r IO) {
	r.Varint32(&data.ActionType)
	r.Varint32(&data.HotBarSlot)
	r.ItemInstance(&data.HeldItem)
	r.Vec3(&data.HeadPosition)
}

func (*NormalTransactionData) Marshal(IO) {}

func (*MismatchTransactionData) Marshal(IO) {}

type LegacySetItemSlot struct {
	ContainerID byte
	Slots       []byte
}

func (x *LegacySetItemSlot) Marshal(r IO) {
	r.Uint8(&x.ContainerID)
	r.ByteSlice(&x.Slots)
}
