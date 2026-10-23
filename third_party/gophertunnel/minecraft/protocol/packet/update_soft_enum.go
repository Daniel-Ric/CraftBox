package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	SoftEnumActionAdd = iota
	SoftEnumActionRemove
	SoftEnumActionSet
)

type UpdateSoftEnum struct {
	EnumType string

	Options []string

	ActionType byte
}

func (*UpdateSoftEnum) ID() uint32 {
	return IDUpdateSoftEnum
}

func (pk *UpdateSoftEnum) Marshal(io protocol.IO) {
	io.String(&pk.EnumType)
	protocol.FuncSlice(io, &pk.Options, io.String)
	io.Uint8(&pk.ActionType)
}
