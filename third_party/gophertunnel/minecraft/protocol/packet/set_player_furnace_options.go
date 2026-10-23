package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	FurnaceTypeNone = iota
	FurnaceTypeFurnace
	FurnaceTypeBlastFurnace
	FurnaceTypeSmoker
)

type SetPlayerFurnaceOptions struct {
	FurnaceType byte

	FurnaceOptions protocol.FurnaceOptions
}

func (*SetPlayerFurnaceOptions) ID() uint32 {
	return IDSetPlayerFurnaceOptions
}

func (pk *SetPlayerFurnaceOptions) Marshal(io protocol.IO) {
	io.Uint8(&pk.FurnaceType)
	protocol.Single(io, &pk.FurnaceOptions)
}
