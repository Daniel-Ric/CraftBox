package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AddPlayer struct {
	UUID uuid.UUID

	Username string

	EntityRuntimeID uint64

	PlatformChatID string

	Position mgl32.Vec3

	Velocity mgl32.Vec3

	Pitch float32

	Yaw float32

	HeadYaw float32

	HeldItem protocol.ItemInstance

	GameType int32

	EntityMetadata protocol.EntityMetadata

	EntityProperties protocol.EntityProperties

	AbilityData protocol.AbilityData

	EntityLinks []protocol.EntityLink

	DeviceID string

	BuildPlatform int32
}

func (*AddPlayer) ID() uint32 {
	return IDAddPlayer
}

func (pk *AddPlayer) Marshal(io protocol.IO) {
	io.UUID(&pk.UUID)
	io.String(&pk.Username)
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.String(&pk.PlatformChatID)
	io.Vec3(&pk.Position)
	io.Vec3(&pk.Velocity)
	io.Float32(&pk.Pitch)
	io.Float32(&pk.Yaw)
	io.Float32(&pk.HeadYaw)
	io.ItemInstance(&pk.HeldItem)
	io.Varint32(&pk.GameType)
	io.EntityMetadata(&pk.EntityMetadata)
	protocol.Single(io, &pk.EntityProperties)
	protocol.Single(io, &pk.AbilityData)
	protocol.Slice(io, &pk.EntityLinks)
	io.String(&pk.DeviceID)
	io.Int32(&pk.BuildPlatform)
}
