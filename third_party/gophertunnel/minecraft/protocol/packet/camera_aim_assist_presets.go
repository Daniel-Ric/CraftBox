package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	CameraAunAssistPresetOperationSet = iota
	CameraAunAssistPresetOperationAddToExisting
)

type CameraAimAssistPresets struct {
	Categories []protocol.CameraAimAssistCategory

	Presets []protocol.CameraAimAssistPreset

	Operation byte
}

func (*CameraAimAssistPresets) ID() uint32 {
	return IDCameraAimAssistPresets
}

func (pk *CameraAimAssistPresets) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Categories)
	protocol.Slice(io, &pk.Presets)
	io.Uint8(&pk.Operation)
}
