package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientBoundCloseForm struct{}

func (*ClientBoundCloseForm) ID() uint32 {
	return IDClientBoundCloseForm
}

func (pk *ClientBoundCloseForm) Marshal(protocol.IO) {}
