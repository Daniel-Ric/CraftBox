package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type MotionPredictionHints struct {
	EntityRuntimeID uint64

	Velocity mgl32.Vec3

	OnGround bool
}

func (*MotionPredictionHints) ID() uint32 {
	return IDMotionPredictionHints
}

func (pk *MotionPredictionHints) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.Vec3(&pk.Velocity)
	io.Bool(&pk.OnGround)
}
