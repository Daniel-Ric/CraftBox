package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type RemoveVolumeEntity struct {
	EntityRuntimeID uint32

	Dimension int32
}

func (*RemoveVolumeEntity) ID() uint32 {
	return IDRemoveVolumeEntity
}

func (pk *RemoveVolumeEntity) Marshal(io protocol.IO) {
	io.ActorRuntimeIDVaruint32(&pk.EntityRuntimeID)
	io.Varint32(&pk.Dimension)
}
