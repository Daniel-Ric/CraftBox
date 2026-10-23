package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	BlockToEntityTransition = iota + 1
	EntityToBlockTransition
)

type UpdateBlockSynced struct {
	Position protocol.BlockPos

	NewBlockRuntimeID uint32

	Flags uint32

	Layer uint32

	EntityUniqueID uint64

	TransitionType uint64
}

func (*UpdateBlockSynced) ID() uint32 {
	return IDUpdateBlockSynced
}

func (pk *UpdateBlockSynced) Marshal(io protocol.IO) {
	io.BlockPos(&pk.Position)
	io.Varuint32(&pk.NewBlockRuntimeID)
	io.Varuint32(&pk.Flags)
	io.Varuint32(&pk.Layer)
	io.ActorUniqueIDVaruint64(&pk.EntityUniqueID)
	io.Varuint64(&pk.TransitionType)
}
