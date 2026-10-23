package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type CommandRequest struct {
	CommandLine string

	CommandOrigin protocol.CommandOrigin

	Internal bool

	Version string
}

func (*CommandRequest) ID() uint32 {
	return IDCommandRequest
}

func (pk *CommandRequest) Marshal(io protocol.IO) {
	io.String(&pk.CommandLine)
	protocol.CommandOriginData(io, &pk.CommandOrigin)
	io.Bool(&pk.Internal)
	io.String(&pk.Version)
}
