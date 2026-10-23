package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientBoundDataDrivenUICloseScreen struct {
	FormID protocol.Optional[uint32]
}

func (*ClientBoundDataDrivenUICloseScreen) ID() uint32 {
	return IDClientBoundDataDrivenUICloseScreen
}

func (pk *ClientBoundDataDrivenUICloseScreen) Marshal(io protocol.IO) {
	protocol.OptionalFunc(io, &pk.FormID, io.Uint32)
}
