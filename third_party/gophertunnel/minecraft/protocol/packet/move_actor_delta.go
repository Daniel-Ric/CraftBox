package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type MoveActorDelta struct {
	EntityRuntimeID uint64

	PositionX protocol.Optional[float32]
	PositionY protocol.Optional[float32]
	PositionZ protocol.Optional[float32]

	RotationX     protocol.Optional[float32]
	RotationY     protocol.Optional[float32]
	RotationYHead protocol.Optional[float32]

	OnGround bool

	ForceMove bool

	ForceMoveLocalEntity bool

	ForceCompletion bool

	Ticks uint64
}

func (*MoveActorDelta) ID() uint32 {
	return IDMoveActorDelta
}

func (pk *MoveActorDelta) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	protocol.OptionalFunc(io, &pk.PositionX, io.Float32)
	protocol.OptionalFunc(io, &pk.PositionY, io.Float32)
	protocol.OptionalFunc(io, &pk.PositionZ, io.Float32)
	protocol.OptionalFunc(io, &pk.RotationX, io.ByteFloat)
	protocol.OptionalFunc(io, &pk.RotationY, io.ByteFloat)
	protocol.OptionalFunc(io, &pk.RotationYHead, io.ByteFloat)
	io.Bool(&pk.OnGround)
	io.Bool(&pk.ForceMove)
	io.Bool(&pk.ForceMoveLocalEntity)
	io.Bool(&pk.ForceCompletion)
	io.Varuint64(&pk.Ticks)
}
