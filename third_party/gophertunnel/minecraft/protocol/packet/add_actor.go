package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AddActor struct {
	EntityUniqueID int64

	EntityRuntimeID uint64

	EntityType string

	Position mgl32.Vec3

	Velocity mgl32.Vec3

	Pitch float32

	Yaw float32

	HeadYaw float32

	BodyYaw float32

	Attributes []protocol.AttributeValue

	EntityMetadata protocol.EntityMetadata

	EntityProperties protocol.EntityProperties

	EntityLinks []protocol.EntityLink
}

func (*AddActor) ID() uint32 {
	return IDAddActor
}

func (pk *AddActor) Marshal(io protocol.IO) {
	io.ActorUniqueID(&pk.EntityUniqueID)
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.String(&pk.EntityType)
	io.Vec3(&pk.Position)
	io.Vec3(&pk.Velocity)
	io.Float32(&pk.Pitch)
	io.Float32(&pk.Yaw)
	io.Float32(&pk.HeadYaw)
	io.Float32(&pk.BodyYaw)
	protocol.Slice(io, &pk.Attributes)
	io.EntityMetadata(&pk.EntityMetadata)
	protocol.Single(io, &pk.EntityProperties)
	protocol.Slice(io, &pk.EntityLinks)
}
