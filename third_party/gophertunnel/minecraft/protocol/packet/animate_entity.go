package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AnimateEntity struct {
	Animation string

	NextState string

	StopCondition string

	StopConditionVersion int32

	Controller string

	BlendOutTime float32

	EntityRuntimeIDs []uint64
}

func (*AnimateEntity) ID() uint32 {
	return IDAnimateEntity
}

func (pk *AnimateEntity) Marshal(io protocol.IO) {
	io.String(&pk.Animation)
	io.String(&pk.NextState)
	io.String(&pk.StopCondition)
	io.Int32(&pk.StopConditionVersion)
	io.String(&pk.Controller)
	io.Float32(&pk.BlendOutTime)
	protocol.FuncSlice(io, &pk.EntityRuntimeIDs, io.ActorRuntimeID)
}
