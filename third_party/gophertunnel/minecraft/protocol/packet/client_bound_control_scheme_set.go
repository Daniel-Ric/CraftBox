package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ControlSchemeLockedPlayerRelativeStrafe = iota
	ControlSchemeCameraRelative
	ControlSchemeCameraRelativeStrafe
	ControlSchemePlayerRelative
	ControlSchemePlayerRelativeStrafe
)

type ClientBoundControlSchemeSet struct {
	ControlScheme byte
}

func (*ClientBoundControlSchemeSet) ID() uint32 {
	return IDClientBoundControlSchemeSet
}

func (pk *ClientBoundControlSchemeSet) Marshal(io protocol.IO) {
	io.Uint8(&pk.ControlScheme)
}
