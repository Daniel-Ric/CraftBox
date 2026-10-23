package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type CreativeContent struct {
	Groups []protocol.CreativeGroup

	Items []protocol.CreativeItem
}

func (*CreativeContent) ID() uint32 {
	return IDCreativeContent
}

func (pk *CreativeContent) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Groups)
	protocol.Slice(io, &pk.Items)
}
