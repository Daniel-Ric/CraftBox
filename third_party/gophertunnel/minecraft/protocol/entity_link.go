package protocol

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
