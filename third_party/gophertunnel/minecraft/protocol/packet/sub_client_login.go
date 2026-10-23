package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SubClientLogin struct {
	ConnectionRequest []byte
}

func (*SubClientLogin) ID() uint32 {
	return IDSubClientLogin
}

func (pk *SubClientLogin) Marshal(io protocol.IO) {
	io.ByteSlice(&pk.ConnectionRequest)
}
