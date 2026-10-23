package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	GameTypeSurvival = iota
	GameTypeCreative
	GameTypeAdventure
	_
	_
	GameTypeDefault
	GameTypeSpectator
)

type SetPlayerGameType struct {
	GameType int32
}

func (*SetPlayerGameType) ID() uint32 {
	return IDSetPlayerGameType
}

func (pk *SetPlayerGameType) Marshal(io protocol.IO) {
	io.Varint32(&pk.GameType)
}
