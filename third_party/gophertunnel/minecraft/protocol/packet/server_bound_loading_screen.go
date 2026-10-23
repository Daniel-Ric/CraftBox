package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	LoadingScreenTypeUnknown = iota
	LoadingScreenTypeStart
	LoadingScreenTypeEnd
)

type ServerBoundLoadingScreen struct {
	Type int32

	LoadingScreenID protocol.Optional[uint32]
}

func (*ServerBoundLoadingScreen) ID() uint32 {
	return IDServerBoundLoadingScreen
}

func (pk *ServerBoundLoadingScreen) Marshal(io protocol.IO) {
	io.Varint32(&pk.Type)
	protocol.OptionalFunc(io, &pk.LoadingScreenID, io.Uint32)
}
