package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ItemStackResponse struct {
	Responses []protocol.ItemStackResponse
}

func (*ItemStackResponse) ID() uint32 {
	return IDItemStackResponse
}

func (pk *ItemStackResponse) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Responses)
}
