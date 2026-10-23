package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientBoundDataDrivenUIShowScreen struct {
	ScreenID string

	FormID uint32

	DataInstanceID protocol.Optional[uint32]
}

func (*ClientBoundDataDrivenUIShowScreen) ID() uint32 {
	return IDClientBoundDataDrivenUIShowScreen
}

func (pk *ClientBoundDataDrivenUIShowScreen) Marshal(io protocol.IO) {
	io.String(&pk.ScreenID)
	io.Uint32(&pk.FormID)
	protocol.OptionalFunc(io, &pk.DataInstanceID, io.Uint32)
}
