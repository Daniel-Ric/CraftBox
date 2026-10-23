package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetCommandsEnabled struct {
	Enabled bool
}

func (*SetCommandsEnabled) ID() uint32 {
	return IDSetCommandsEnabled
}

func (pk *SetCommandsEnabled) Marshal(io protocol.IO) {
	io.Bool(&pk.Enabled)
}
