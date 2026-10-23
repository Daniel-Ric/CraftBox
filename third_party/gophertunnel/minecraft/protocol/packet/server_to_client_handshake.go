package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ServerToClientHandshake struct {
	JWT []byte
}

func (*ServerToClientHandshake) ID() uint32 {
	return IDServerToClientHandshake
}

func (pk *ServerToClientHandshake) Marshal(io protocol.IO) {
	io.ByteSlice(&pk.JWT)
}
