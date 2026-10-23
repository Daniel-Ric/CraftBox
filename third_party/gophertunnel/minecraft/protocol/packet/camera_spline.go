package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type CameraSpline struct {
	Splines []protocol.CameraSplineDefinition
}

func (*CameraSpline) ID() uint32 {
	return IDCameraSpline
}

func (pk *CameraSpline) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Splines)
}
