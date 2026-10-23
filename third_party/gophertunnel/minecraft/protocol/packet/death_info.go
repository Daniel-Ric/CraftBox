package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type DeathInfo struct {
	Cause string

	Messages []string
}

func (*DeathInfo) ID() uint32 {
	return IDDeathInfo
}

func (pk *DeathInfo) Marshal(io protocol.IO) {
	io.String(&pk.Cause)
	protocol.FuncSlice(io, &pk.Messages, io.String)
}
