package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ChangeMobProperty struct {
	EntityUniqueID int64

	Property string

	BoolValue bool

	StringValue string

	IntValue int32

	FloatValue float32
}

func (*ChangeMobProperty) ID() uint32 {
	return IDChangeMobProperty
}

func (pk *ChangeMobProperty) Marshal(io protocol.IO) {
	io.ActorUniqueID(&pk.EntityUniqueID)
	io.String(&pk.Property)
	io.Bool(&pk.BoolValue)
	io.String(&pk.StringValue)
	io.Varint32(&pk.IntValue)
	io.Float32(&pk.FloatValue)
}
