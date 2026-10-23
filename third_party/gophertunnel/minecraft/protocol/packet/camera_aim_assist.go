package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	CameraAimAssistActionSet = iota
	CameraAimAssistActionClear
)

type CameraAimAssist struct {
	Preset string

	Angle mgl32.Vec2

	Distance float32

	TargetMode byte

	Action byte

	ShowDebugRender bool
}

func (*CameraAimAssist) ID() uint32 {
	return IDCameraAimAssist
}

func (pk *CameraAimAssist) Marshal(io protocol.IO) {
	io.String(&pk.Preset)
	io.Vec2(&pk.Angle)
	io.Float32(&pk.Distance)
	io.Uint8(&pk.TargetMode)
	io.Uint8(&pk.Action)
	io.Bool(&pk.ShowDebugRender)
}
