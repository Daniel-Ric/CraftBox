package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	BlockUpdateNeighbours = 1 << iota
	BlockUpdateNetwork
	BlockUpdateNoGraphics
	BlockUpdatePriority
)

type UpdateBlock struct {
	Position protocol.BlockPos

	NewBlockRuntimeID uint32

	Flags uint32

	Layer uint32
}

func (*UpdateBlock) ID() uint32 {
	return IDUpdateBlock
}

func (pk *UpdateBlock) Marshal(io protocol.IO) {
	io.BlockPos(&pk.Position)
	io.Varuint32(&pk.NewBlockRuntimeID)
	io.Varuint32(&pk.Flags)
	io.Varuint32(&pk.Layer)
}
