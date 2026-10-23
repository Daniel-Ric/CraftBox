package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type Camera struct {
	CameraEntityUniqueID int64

	TargetPlayerUniqueID int64
}

func (*Camera) ID() uint32 {
	return IDCamera
}

func (pk *Camera) Marshal(io protocol.IO) {
	io.ActorUniqueID(&pk.CameraEntityUniqueID)
	io.ActorUniqueID(&pk.TargetPlayerUniqueID)
}
