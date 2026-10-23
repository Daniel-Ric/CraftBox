package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ScriptCustomEvent struct {
	EventName string

	EventData []byte
}

func (*ScriptCustomEvent) ID() uint32 {
	return IDScriptCustomEvent
}

func (pk *ScriptCustomEvent) Marshal(io protocol.IO) {
	io.String(&pk.EventName)
	io.ByteSlice(&pk.EventData)
}
