package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	MoveFlagOnGround = 1 << iota
	MoveFlagTeleport
	MoveFlagForceMove
	MoveFlagForceCompletion
)

type MoveActorAbsolute struct {
	EntityRuntimeID uint64

	Flags byte

	Position mgl32.Vec3

	Rotation mgl32.Vec3
}

func (*MoveActorAbsolute) ID() uint32 {
	return IDMoveActorAbsolute
}

func (pk *MoveActorAbsolute) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.Uint8(&pk.Flags)
	io.Vec3(&pk.Position)
	io.ByteFloat(&pk.Rotation[0])
	io.ByteFloat(&pk.Rotation[1])
	io.ByteFloat(&pk.Rotation[2])
}
