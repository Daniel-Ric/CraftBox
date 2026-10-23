package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	CommandBlockImpulse = iota
	CommandBlockRepeating
	CommandBlockChain
)

type CommandBlockUpdate struct {
	Block bool

	Position protocol.BlockPos

	Mode uint32

	NeedsRedstone bool

	Conditional bool

	MinecartEntityRuntimeID uint64

	Command string

	LastOutput string

	Name string

	FilteredName string

	ShouldTrackOutput bool

	TickDelay uint32

	ExecuteOnFirstTick bool
}

func (*CommandBlockUpdate) ID() uint32 {
	return IDCommandBlockUpdate
}

func (pk *CommandBlockUpdate) Marshal(io protocol.IO) {
	io.Bool(&pk.Block)
	if pk.Block {
		io.BlockPos(&pk.Position)
		io.Varuint32(&pk.Mode)
		io.Bool(&pk.NeedsRedstone)
		io.Bool(&pk.Conditional)
	} else {
		io.ActorRuntimeID(&pk.MinecartEntityRuntimeID)
	}
	io.String(&pk.Command)
	io.String(&pk.LastOutput)
	io.String(&pk.Name)
	io.String(&pk.FilteredName)
	io.Bool(&pk.ShouldTrackOutput)
	io.Uint32(&pk.TickDelay)
	io.Bool(&pk.ExecuteOnFirstTick)
}
