package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	InputFlagAscend = iota
	InputFlagDescend
	InputFlagNorthJump
	InputFlagJumpDown
	InputFlagSprintDown
	InputFlagChangeHeight
	InputFlagJumping
	InputFlagAutoJumpingInWater
	InputFlagSneaking
	InputFlagSneakDown
	InputFlagUp
	InputFlagDown
	InputFlagLeft
	InputFlagRight
	InputFlagUpLeft
	InputFlagUpRight
	InputFlagWantUp
	InputFlagWantDown
	InputFlagWantDownSlow
	InputFlagWantUpSlow
	InputFlagSprinting
	InputFlagAscendBlock
	InputFlagDescendBlock
	InputFlagSneakToggleDown
	InputFlagPersistSneak
	InputFlagStartSprinting
	InputFlagStopSprinting
	InputFlagStartSneaking
	InputFlagStopSneaking
	InputFlagStartSwimming
	InputFlagStopSwimming
	InputFlagStartJumping
	InputFlagStartGliding
	InputFlagStopGliding
	InputFlagPerformItemInteraction
	InputFlagPerformBlockActions
	InputFlagPerformItemStackRequest
	InputFlagHandledTeleport
	InputFlagEmoting
	InputFlagMissedSwing
	InputFlagStartCrawling
	InputFlagStopCrawling
	InputFlagStartFlying
	InputFlagStopFlying
	InputFlagClientAckServerData
	InputFlagClientPredictedVehicle
	InputFlagPaddlingLeft
	InputFlagPaddlingRight
	InputFlagBlockBreakingDelayEnabled
	InputFlagHorizontalCollision
	InputFlagVerticalCollision
	InputFlagDownLeft
	InputFlagDownRight
	InputFlagStartUsingItem
	InputFlagCameraRelativeMovementEnabled
	InputFlagRotControlledByMoveDirection
	InputFlagStartSpinAttack
	InputFlagStopSpinAttack
	InputFlagIsHotbarTouchOnly
	InputFlagJumpReleasedRaw
	InputFlagJumpPressedRaw
	InputFlagJumpCurrentRaw
	InputFlagSneakReleasedRaw
	InputFlagSneakPressedRaw
	InputFlagSneakCurrentRaw
	InputFlagInternalUpdate

	InputFlagCount
)

const (
	InputModeUndefined = iota
	InputModeMouse
	InputModeTouch
	InputModeGamePad
	InputModeMotionController
	InputModeCount
)

const (
	PlayModeNormal = iota
	PlayModeTeaser
	PlayModeScreen
	PlayModeViewer
	PlayModeReality
	PlayModePlacement
	PlayModeLivingRoom
	PlayModeExitLevel
	PlayModeExitLevelLivingRoom
)

const PlayModeNumModes = PlayModeExitLevelLivingRoom + 1

const (
	InteractionModelTouch = iota
	InteractionModelCrosshair
	InteractionModelClassic
	InteractionModelCount
)

type PlayerAuthInput struct {
	Pitch, Yaw float32

	Position mgl32.Vec3

	MoveVector mgl32.Vec2

	HeadYaw float32

	InputData protocol.InputFlags

	InputMode uint32

	PlayMode uint32

	InteractionModel int32

	InteractPitch, InteractYaw float32

	Tick uint64

	Delta mgl32.Vec3

	ItemInteractionData protocol.Optional[protocol.UseItemTransactionData]

	ItemStackRequest protocol.Optional[protocol.ItemStackRequest]

	BlockActions protocol.Optional[[]protocol.PlayerBlockAction]

	VehicleRotation protocol.Optional[mgl32.Vec2]

	ClientPredictedVehicle protocol.Optional[int64]

	AnalogueMoveVector mgl32.Vec2

	CameraOrientation mgl32.Vec3

	RawMoveVector mgl32.Vec2
}

func (pk *PlayerAuthInput) ID() uint32 {
	return IDPlayerAuthInput
}

func (pk *PlayerAuthInput) Marshal(io protocol.IO) {
	io.Float32(&pk.Pitch)
	io.Float32(&pk.Yaw)
	io.Vec3(&pk.Position)
	io.Vec2(&pk.MoveVector)
	io.Float32(&pk.HeadYaw)
	protocol.InputFlagList(io, &pk.InputData, InputFlagCount)
	io.Varuint32(&pk.InputMode)
	io.Varuint32(&pk.PlayMode)
	io.Varint32(&pk.InteractionModel)
	io.Float32(&pk.InteractPitch)
	io.Float32(&pk.InteractYaw)
	io.Varuint64(&pk.Tick)
	io.Vec3(&pk.Delta)
	protocol.OptionalFunc(io, &pk.ItemInteractionData, io.PlayerInventoryAction)
	protocol.OptionalFunc(io, &pk.ItemStackRequest, func(x *protocol.ItemStackRequest) {
		x.Marshal(io)
	})
	protocol.OptionalFunc(io, &pk.BlockActions, func(x *[]protocol.PlayerBlockAction) {
		protocol.Slice(io, x)
	})
	protocol.OptionalFunc(io, &pk.VehicleRotation, io.Vec2)
	protocol.OptionalFunc(io, &pk.ClientPredictedVehicle, io.ActorUniqueID)
	io.Vec2(&pk.AnalogueMoveVector)
	io.Vec3(&pk.CameraOrientation)
	io.Vec2(&pk.RawMoveVector)
}
