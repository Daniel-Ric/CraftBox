package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type CameraInstruction struct {
	Set protocol.Optional[protocol.CameraInstructionSet]

	Clear protocol.Optional[bool]

	Fade protocol.Optional[protocol.CameraInstructionFade]

	Target protocol.Optional[protocol.CameraInstructionTarget]

	RemoveTarget protocol.Optional[bool]

	FieldOfView protocol.Optional[protocol.CameraInstructionFieldOfView]

	Spline protocol.Optional[protocol.CameraSplineInstruction]

	AttachToEntity protocol.Optional[int64]

	DetachFromEntity protocol.Optional[bool]
}

func (*CameraInstruction) ID() uint32 {
	return IDCameraInstruction
}

func (pk *CameraInstruction) Marshal(io protocol.IO) {
	protocol.OptionalMarshaler(io, &pk.Set)
	protocol.OptionalFunc(io, &pk.Clear, io.Bool)
	protocol.OptionalMarshaler(io, &pk.Fade)
	protocol.OptionalMarshaler(io, &pk.Target)
	protocol.OptionalFunc(io, &pk.RemoveTarget, io.Bool)
	protocol.OptionalMarshaler(io, &pk.FieldOfView)
	protocol.OptionalMarshaler(io, &pk.Spline)
	protocol.OptionalFunc(io, &pk.AttachToEntity, io.ActorUniqueIDInt64)
	protocol.OptionalFunc(io, &pk.DetachFromEntity, io.Bool)
}
