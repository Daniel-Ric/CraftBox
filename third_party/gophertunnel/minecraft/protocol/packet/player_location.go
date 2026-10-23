package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PlayerLocationTypeCoordinates = iota
	PlayerLocationTypeHide
)

type PlayerLocation struct {
	Type int32

	EntityUniqueID int64

	Position mgl32.Vec3
}

func (*PlayerLocation) ID() uint32 {
	return IDPlayerLocation
}

func (pk *PlayerLocation) Marshal(io protocol.IO) {
	io.ActorUniqueID(&pk.EntityUniqueID)
	protocol.IntegerFunc(&pk.Type, io.Varuint32)
	io.Varint32(&pk.Type)

	switch pk.Type {
	case PlayerLocationTypeCoordinates:
		io.Vec3(&pk.Position)
	case PlayerLocationTypeHide:
	default:
		io.UnknownEnumOption(pk.Type, "player location type")
	}
}
