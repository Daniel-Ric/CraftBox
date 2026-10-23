package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AvailableCommands struct {
	EnumValues []string

	ChainedSubcommandValues []string

	Suffixes []string

	Enums []protocol.CommandEnum

	ChainedSubcommands []protocol.ChainedSubcommand

	Commands []protocol.Command

	DynamicEnums []protocol.DynamicEnum

	Constraints []protocol.CommandEnumConstraint
}

func (*AvailableCommands) ID() uint32 {
	return IDAvailableCommands
}

func (pk *AvailableCommands) Marshal(io protocol.IO) {
	protocol.FuncSlice(io, &pk.EnumValues, io.String)
	protocol.FuncSlice(io, &pk.ChainedSubcommandValues, io.String)
	protocol.FuncSlice(io, &pk.Suffixes, io.String)
	protocol.FuncIOSlice(io, &pk.Enums, protocol.CommandEnumContext{EnumValues: pk.EnumValues}.Marshal)
	protocol.Slice(io, &pk.ChainedSubcommands)
	protocol.Slice(io, &pk.Commands)
	protocol.Slice(io, &pk.DynamicEnums)
	protocol.Slice(io, &pk.Constraints)
}
