package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	_ = iota + 1
	_
	InteractActionLeaveVehicle
	InteractActionMouseOverEntity
	InteractActionNPCOpen
	InteractActionOpenInventory
)

type Interact struct {
	ActionType byte

	TargetEntityRuntimeID uint64

	Position protocol.Optional[mgl32.Vec3]
}

func (*Interact) ID() uint32 {
	return IDInteract
}

func (pk *Interact) Marshal(io protocol.IO) {
	io.Uint8(&pk.ActionType)
	io.ActorRuntimeID(&pk.TargetEntityRuntimeID)
	protocol.OptionalFunc(io, &pk.Position, io.Vec3)
}
