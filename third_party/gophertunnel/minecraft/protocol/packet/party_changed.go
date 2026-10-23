package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PartyChanged struct {
	PartyInfo protocol.Optional[PartyInfo]
}

func (*PartyChanged) ID() uint32 {
	return IDPartyChanged
}

func (pk *PartyChanged) Marshal(io protocol.IO) {
	protocol.OptionalMarshaler(io, &pk.PartyInfo)
}

type PartyInfo struct {
	PartyID string

	PartyLeader bool
}

func (x *PartyInfo) Marshal(io protocol.IO) {
	io.String(&x.PartyID)
	io.Bool(&x.PartyLeader)
}
