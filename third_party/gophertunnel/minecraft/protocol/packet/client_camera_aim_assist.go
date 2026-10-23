package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ClientCameraAimAssistActionSet = iota
	ClientCameraAimAssistActionClear
)

type ClientCameraAimAssist struct {
	PresetID string

	Action byte

	AllowAimAssist bool
}

func (*ClientCameraAimAssist) ID() uint32 {
	return IDClientCameraAimAssist
}

func (pk *ClientCameraAimAssist) Marshal(io protocol.IO) {
	io.String(&pk.PresetID)
	io.Uint8(&pk.Action)
	io.Bool(&pk.AllowAimAssist)
}
