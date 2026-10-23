package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AutomationClientConnect struct {
	ServerURI string
}

func (*AutomationClientConnect) ID() uint32 {
	return IDAutomationClientConnect
}

func (pk *AutomationClientConnect) Marshal(io protocol.IO) {
	io.String(&pk.ServerURI)
}
