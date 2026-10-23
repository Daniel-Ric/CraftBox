package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	MoveModeNormal = iota
	MoveModeReset
	MoveModeTeleport
	MoveModeRotation
)

const (
	TeleportCauseUnknown = iota
	TeleportCauseProjectile
	TeleportCauseChorusFruit
	TeleportCauseCommand
	TeleportCauseBehaviour
)

type MovePlayer struct {
	EntityRuntimeID uint64

	Position mgl32.Vec3

	Pitch float32

	Yaw float32

	HeadYaw float32

	Mode byte

	OnGround bool

	RiddenEntityRuntimeID uint64
	TeleportData          protocol.Optional[protocol.TeleportData]

	Tick uint64
}

func (*MovePlayer) ID() uint32 {
	return IDMovePlayer
}

func (pk *MovePlayer) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.Vec3(&pk.Position)
	io.Float32(&pk.Pitch)
	io.Float32(&pk.Yaw)
	io.Float32(&pk.HeadYaw)
	io.Uint8(&pk.Mode)
	io.Bool(&pk.OnGround)
	io.ActorRuntimeID(&pk.RiddenEntityRuntimeID)
	protocol.OptionalMarshaler(io, &pk.TeleportData)
	io.Varuint64(&pk.Tick)
}
