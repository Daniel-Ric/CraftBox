package protocol

import "github.com/go-gl/mathgl/mgl32"

const (
	EntityLinkRemove = iota

	EntityLinkRider

	EntityLinkPassenger
)

type EntityLink struct {
	RiddenEntityUniqueID int64

	RiderEntityUniqueID int64

	Type byte

	Immediate bool

	RiderInitiated bool

	VehicleAngularVelocity float32
}

func (x *EntityLink) Marshal(r IO) {
	r.ActorUniqueID(&x.RiddenEntityUniqueID)
	r.ActorUniqueID(&x.RiderEntityUniqueID)
	r.Uint8(&x.Type)
	r.Bool(&x.Immediate)
	r.Bool(&x.RiderInitiated)
	r.Float32(&x.VehicleAngularVelocity)
}

const (
	PassengerOfBlockEmoteStanding = iota
	PassengerOfBlockEmoteRiding
	PassengerOfBlockEmoteLaying
)

type PassengerOfBlockData struct {
	BlockPosition BlockPos

	Offset mgl32.Vec3

	Rotation float32

	RotationLimit float32

	EmoteType uint8
}

func (x *PassengerOfBlockData) Marshal(r IO) {
	r.BlockPos(&x.BlockPosition)
	r.Vec3(&x.Offset)
	r.Float32(&x.Rotation)
	r.Float32(&x.RotationLimit)
	r.Uint8(&x.EmoteType)
}
