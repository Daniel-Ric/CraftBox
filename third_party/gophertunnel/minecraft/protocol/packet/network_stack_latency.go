package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type NetworkStackLatency struct {
	Timestamp int64

	NeedsResponse bool
}

func (*NetworkStackLatency) ID() uint32 {
	return IDNetworkStackLatency
}

func (pk *NetworkStackLatency) Marshal(io protocol.IO) {
	io.Int64(&pk.Timestamp)
	io.Bool(&pk.NeedsResponse)
}
