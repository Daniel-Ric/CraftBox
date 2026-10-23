package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type Login struct {
	ClientProtocol int32

	ConnectionRequest []byte
}

func (*Login) ID() uint32 {
	return IDLogin
}

func (pk *Login) Marshal(io protocol.IO) {
	io.BEInt32(&pk.ClientProtocol)
	io.ByteSlice(&pk.ConnectionRequest)
}
