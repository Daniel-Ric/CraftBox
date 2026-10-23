package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type CameraPresets struct {
	Presets []protocol.CameraPreset
}

func (*CameraPresets) ID() uint32 {
	return IDCameraPresets
}

func (pk *CameraPresets) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Presets)
}
