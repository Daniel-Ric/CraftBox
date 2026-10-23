package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type CameraAimAssistActorPriority struct {
	PriorityData []protocol.CameraAimAssistActorPriorityData
}

func (*CameraAimAssistActorPriority) ID() uint32 {
	return IDCameraAimAssistActorPriority
}

func (pk *CameraAimAssistActorPriority) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.PriorityData)
}
