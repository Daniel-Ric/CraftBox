package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientCacheStatus struct {
	Enabled bool
}

func (pk *ClientCacheStatus) ID() uint32 {
	return IDClientCacheStatus
}

func (pk *ClientCacheStatus) Marshal(io protocol.IO) {
	io.Bool(&pk.Enabled)
}
