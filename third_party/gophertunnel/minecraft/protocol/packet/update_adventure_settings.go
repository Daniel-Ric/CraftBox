package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type UpdateAdventureSettings struct {
	NoPvM bool

	NoMvP bool

	ImmutableWorld bool

	ShowNameTags bool

	AutoJump bool
}

func (*UpdateAdventureSettings) ID() uint32 {
	return IDUpdateAdventureSettings
}

func (pk *UpdateAdventureSettings) Marshal(io protocol.IO) {
	io.Bool(&pk.NoPvM)
	io.Bool(&pk.NoMvP)
	io.Bool(&pk.ImmutableWorld)
	io.Bool(&pk.ShowNameTags)
	io.Bool(&pk.AutoJump)
}
