package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientToServerHandshake struct {
}

func (*ClientToServerHandshake) ID() uint32 {
	return IDClientToServerHandshake
}

func (*ClientToServerHandshake) Marshal(protocol.IO) {}
