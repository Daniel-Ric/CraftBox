package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PlayerUpdateEntityOverridesTypeClearAll = iota
	PlayerUpdateEntityOverridesTypeRemove
	PlayerUpdateEntityOverridesTypeInt
	PlayerUpdateEntityOverridesTypeFloat
)

type PlayerUpdateEntityOverrides struct {
	EntityUniqueID int64

	PropertyIndex uint32

	Type uint32

	IntValue int32

	FloatValue float32
}

func (*PlayerUpdateEntityOverrides) ID() uint32 {
	return IDPlayerUpdateEntityOverrides
}

func (pk *PlayerUpdateEntityOverrides) Marshal(io protocol.IO) {
	io.ActorUniqueID(&pk.EntityUniqueID)
	io.Varuint32(&pk.PropertyIndex)
	io.Varuint32(&pk.Type)
	names := [...]string{"clearoverrides", "removeoverride", "setintoverride", "setfloatoverride"}
	if pk.Type >= uint32(len(names)) {
		io.UnknownEnumOption(pk.Type, "entity override type")
		return
	}
	name := names[pk.Type]
	io.String(&name)
	switch pk.Type {
	case PlayerUpdateEntityOverridesTypeClearAll, PlayerUpdateEntityOverridesTypeRemove:
	case PlayerUpdateEntityOverridesTypeInt:
		io.Int32(&pk.IntValue)
	case PlayerUpdateEntityOverridesTypeFloat:
		io.Float32(&pk.FloatValue)
	}
}
