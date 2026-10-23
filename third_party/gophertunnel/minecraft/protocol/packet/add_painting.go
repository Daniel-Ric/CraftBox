package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AddPainting struct {
	EntityUniqueID int64

	EntityRuntimeID uint64

	Position mgl32.Vec3

	Direction int32

	Title string
}

func (*AddPainting) ID() uint32 {
	return IDAddPainting
}

func (pk *AddPainting) Marshal(io protocol.IO) {
	io.ActorUniqueID(&pk.EntityUniqueID)
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.Vec3(&pk.Position)
	io.Varint32(&pk.Direction)
	io.String(&pk.Title)
}
