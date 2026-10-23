package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

const (
	PartyDestinationCookieIntentNotify = "Notify"
	PartyDestinationCookieIntentOptIn  = "OptIn"
	PartyDestinationCookieIntentOptOut = "OptOut"
)

type SendPartyDestinationCookie struct {
	Cookie string

	Intent string

	DestinationName string
}

func (*SendPartyDestinationCookie) ID() uint32 {
	return IDSendPartyDestinationCookie
}

func (pk *SendPartyDestinationCookie) Marshal(io protocol.IO) {
	io.String(&pk.Cookie)
	io.String(&pk.Intent)
	io.String(&pk.DestinationName)
}
