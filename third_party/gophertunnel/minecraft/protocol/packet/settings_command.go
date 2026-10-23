package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SettingsCommand struct {
	CommandLine string

	SuppressOutput bool
}

func (*SettingsCommand) ID() uint32 {
	return IDSettingsCommand
}

func (pk *SettingsCommand) Marshal(io protocol.IO) {
	io.String(&pk.CommandLine)
	io.Bool(&pk.SuppressOutput)
}
