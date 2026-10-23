package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type PartyDestinationCookieResponse struct {
	Cookie string

	Accepted bool
}

func (*PartyDestinationCookieResponse) ID() uint32 {
	return IDPartyDestinationCookieResponse
}

func (pk *PartyDestinationCookieResponse) Marshal(io protocol.IO) {
	io.String(&pk.Cookie)
	io.Bool(&pk.Accepted)
}
