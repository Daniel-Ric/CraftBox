package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ShowProfile struct {
	XUID string
}

func (*ShowProfile) ID() uint32 {
	return IDShowProfile
}

func (pk *ShowProfile) Marshal(io protocol.IO) {
	io.String(&pk.XUID)
}
