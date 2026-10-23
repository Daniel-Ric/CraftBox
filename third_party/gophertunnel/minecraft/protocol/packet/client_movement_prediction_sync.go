package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientMovementPredictionSync struct {
	ActorFlags protocol.Bitset

	BoundingBoxScale float32

	BoundingBoxWidth float32

	BoundingBoxHeight float32

	MovementSpeed float32

	UnderwaterMovementSpeed float32

	LavaMovementSpeed float32

	JumpStrength float32

	Health float32

	Hunger float32

	FrictionModifier float32

	Bounciness float32

	AirDragModifier float32

	EntityUniqueID int64

	Flying bool
}

func (*ClientMovementPredictionSync) ID() uint32 {
	return IDClientMovementPredictionSync
}

func (pk *ClientMovementPredictionSync) Marshal(io protocol.IO) {
	io.Bitset(&pk.ActorFlags, protocol.EntityDataFlagCount)
	io.Float32(&pk.BoundingBoxScale)
	io.Float32(&pk.BoundingBoxWidth)
	io.Float32(&pk.BoundingBoxHeight)
	io.Float32(&pk.MovementSpeed)
	io.Float32(&pk.UnderwaterMovementSpeed)
	io.Float32(&pk.LavaMovementSpeed)
	io.Float32(&pk.JumpStrength)
	io.Float32(&pk.Health)
	io.Float32(&pk.Hunger)
	io.Float32(&pk.FrictionModifier)
	io.Float32(&pk.Bounciness)
	io.Float32(&pk.AirDragModifier)
	io.ActorUniqueID(&pk.EntityUniqueID)
	io.Bool(&pk.Flying)
}
