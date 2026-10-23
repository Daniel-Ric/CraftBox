package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PlayerAction struct {
	EntityRuntimeID uint64

	ActionType int32

	BlockPosition protocol.BlockPos

	ResultPosition protocol.BlockPos

	BlockFace int32
}

func (*PlayerAction) ID() uint32 {
	return IDPlayerAction
}

func (pk *PlayerAction) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.Varint32(&pk.ActionType)
	io.BlockPos(&pk.BlockPosition)
	io.BlockPos(&pk.ResultPosition)
	io.Varint32(&pk.BlockFace)
}
