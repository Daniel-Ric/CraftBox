package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	BlockEventChangeChestState = 1
)

type BlockEvent struct {
	Position protocol.BlockPos

	EventType int32

	EventData int32
}

func (*BlockEvent) ID() uint32 {
	return IDBlockEvent
}

func (pk *BlockEvent) Marshal(io protocol.IO) {
	io.BlockPos(&pk.Position)
	io.Varint32(&pk.EventType)
	io.Varint32(&pk.EventData)
}
