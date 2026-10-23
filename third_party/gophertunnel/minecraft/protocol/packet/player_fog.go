package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PlayerFog struct {
	Stack []string
}

func (*PlayerFog) ID() uint32 {
	return IDPlayerFog
}

func (pk *PlayerFog) Marshal(io protocol.IO) {
	protocol.FuncSlice(io, &pk.Stack, io.String)
}
