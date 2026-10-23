package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PredictionTypePlayer = iota
	PredictionTypeVehicle
)

type CorrectPlayerMovePrediction struct {
	PredictionType byte

	Position mgl32.Vec3

	Delta mgl32.Vec3

	Rotation mgl32.Vec2

	VehicleAngularVelocity protocol.Optional[float32]

	OnGround bool

	Tick uint64
}

func (*CorrectPlayerMovePrediction) ID() uint32 {
	return IDCorrectPlayerMovePrediction
}

func (pk *CorrectPlayerMovePrediction) Marshal(io protocol.IO) {
	io.Uint8(&pk.PredictionType)
	io.Vec3(&pk.Position)
	io.Vec3(&pk.Delta)
	io.Vec2(&pk.Rotation)
	protocol.OptionalFunc(io, &pk.VehicleAngularVelocity, io.Float32)
	io.Bool(&pk.OnGround)
	io.Varuint64(&pk.Tick)
}
