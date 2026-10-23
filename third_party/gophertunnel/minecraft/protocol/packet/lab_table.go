package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	LabTableActionCombine = iota
	LabTableActionReact
	LabTableActionReset
)

type LabTable struct {
	ActionType byte

	Position protocol.BlockPos

	ReactionType byte
}

func (*LabTable) ID() uint32 {
	return IDLabTable
}

func (pk *LabTable) Marshal(io protocol.IO) {
	io.Uint8(&pk.ActionType)
	io.BlockPos(&pk.Position)
	io.Uint8(&pk.ReactionType)
}
