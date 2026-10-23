package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AddItemActor struct {
	EntityUniqueID int64

	EntityRuntimeID uint64

	Item protocol.ItemInstance

	Position mgl32.Vec3

	Velocity mgl32.Vec3

	EntityMetadata protocol.EntityMetadata

	FromFishing bool
}

func (*AddItemActor) ID() uint32 {
	return IDAddItemActor
}

func (pk *AddItemActor) Marshal(io protocol.IO) {
	io.ActorUniqueID(&pk.EntityUniqueID)
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.ItemInstance(&pk.Item)
	io.Vec3(&pk.Position)
	io.Vec3(&pk.Velocity)
	io.EntityMetadata(&pk.EntityMetadata)
	io.Bool(&pk.FromFishing)
}
