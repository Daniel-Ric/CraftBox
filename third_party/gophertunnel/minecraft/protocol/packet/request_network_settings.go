package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type RequestNetworkSettings struct {
	ClientProtocol int32
}

func (pk *RequestNetworkSettings) ID() uint32 {
	return IDRequestNetworkSettings
}

func (pk *RequestNetworkSettings) Marshal(io protocol.IO) {
	io.BEInt32(&pk.ClientProtocol)
}
