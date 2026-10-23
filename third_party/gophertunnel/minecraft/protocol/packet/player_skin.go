package packet

import (
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PlayerSkin struct {
	UUID uuid.UUID

	Skin protocol.Skin

	NewSkinName string

	OldSkinName string
}

func (*PlayerSkin) ID() uint32 {
	return IDPlayerSkin
}

func (pk *PlayerSkin) Marshal(io protocol.IO) {
	io.UUID(&pk.UUID)
	protocol.Single(io, &pk.Skin)
	io.String(&pk.NewSkinName)
	io.String(&pk.OldSkinName)
}
