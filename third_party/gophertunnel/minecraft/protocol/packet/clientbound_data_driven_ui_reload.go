package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientBoundDataDrivenUIReload struct{}

func (*ClientBoundDataDrivenUIReload) ID() uint32 {
	return IDClientBoundDataDrivenUIReload
}

func (pk *ClientBoundDataDrivenUIReload) Marshal(protocol.IO) {}
