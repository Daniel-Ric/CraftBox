package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

const (
	CameraShakeTypePositional uint8 = iota
	CameraShakeTypeRotational
)

const (
	CameraShakeActionAdd = iota
	CameraShakeActionStop
)

type CameraShake struct {
	Intensity float32

	Duration float32

	Type uint8

	Action uint8
}

func (*CameraShake) ID() uint32 {
	return IDCameraShake
}

func (pk *CameraShake) Marshal(io protocol.IO) {
	io.Float32(&pk.Intensity)
	io.Float32(&pk.Duration)
	io.Uint8(&pk.Type)
	io.Uint8(&pk.Action)
}
