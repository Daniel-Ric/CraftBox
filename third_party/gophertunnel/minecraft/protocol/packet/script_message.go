package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ScriptMessage struct {
	Identifier string

	Data []byte
}

func (pk *ScriptMessage) ID() uint32 {
	return IDScriptMessage
}

func (pk *ScriptMessage) Marshal(io protocol.IO) {
	io.String(&pk.Identifier)
	io.ByteSlice(&pk.Data)
}
