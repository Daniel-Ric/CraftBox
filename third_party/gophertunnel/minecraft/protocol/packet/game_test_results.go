package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type GameTestResults struct {
	Name string

	Succeeded bool

	Error string
}

func (pk *GameTestResults) ID() uint32 {
	return IDGameTestResults
}

func (pk *GameTestResults) Marshal(io protocol.IO) {
	io.Bool(&pk.Succeeded)
	io.String(&pk.Error)
	io.String(&pk.Name)
}
