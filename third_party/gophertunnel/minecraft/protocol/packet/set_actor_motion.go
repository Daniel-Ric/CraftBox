package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetActorMotion struct {
	EntityRuntimeID uint64

	Velocity mgl32.Vec3

	Tick uint64
}

func (*SetActorMotion) ID() uint32 {
	return IDSetActorMotion
}

func (pk *SetActorMotion) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.Vec3(&pk.Velocity)
	io.Varuint64(&pk.Tick)
}
