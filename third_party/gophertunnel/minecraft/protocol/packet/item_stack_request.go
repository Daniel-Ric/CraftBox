package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ItemStackRequest struct {
	Requests []protocol.ItemStackRequest
}

func (*ItemStackRequest) ID() uint32 {
	return IDItemStackRequest
}

func (pk *ItemStackRequest) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Requests)
}
