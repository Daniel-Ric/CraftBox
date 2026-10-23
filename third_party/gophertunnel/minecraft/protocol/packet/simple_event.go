package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	SimpleEventCommandsEnabled = iota + 1
	SimpleEventCommandsDisabled
	SimpleEventUnlockWorldTemplateSettings
)

type SimpleEvent struct {
	EventType uint16
}

func (*SimpleEvent) ID() uint32 {
	return IDSimpleEvent
}

func (pk *SimpleEvent) Marshal(io protocol.IO) {
	io.Uint16(&pk.EventType)
}
