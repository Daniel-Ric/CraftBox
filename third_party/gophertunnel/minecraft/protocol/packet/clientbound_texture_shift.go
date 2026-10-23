package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	TextureShiftActionInvalid = iota
	TextureShiftActionInitialize
	TextureShiftActionStart
	TextureShiftActionSetEnabled
	TextureShiftActionSync
)

type ClientBoundTextureShift struct {
	ActionID uint8

	CollectionName string

	FromStep string

	ToStep string

	AllSteps []string

	CurrentLengthTicks uint64

	TotalLengthTicks uint64

	Enabled bool
}

func (*ClientBoundTextureShift) ID() uint32 {
	return IDClientBoundTextureShift
}

func (pk *ClientBoundTextureShift) Marshal(io protocol.IO) {
	io.Uint8(&pk.ActionID)
	io.String(&pk.CollectionName)
	io.String(&pk.FromStep)
	io.String(&pk.ToStep)
	protocol.FuncSlice(io, &pk.AllSteps, io.String)
	io.Varuint64(&pk.CurrentLengthTicks)
	io.Varuint64(&pk.TotalLengthTicks)
	io.Bool(&pk.Enabled)
}
