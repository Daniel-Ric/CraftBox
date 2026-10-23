package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	EnableMultiPlayer = iota
	DisableMultiPlayer
	RefreshJoinCode
)

type MultiPlayerSettings struct {
	ActionType int32
}

func (*MultiPlayerSettings) ID() uint32 {
	return IDMultiPlayerSettings
}

func (pk *MultiPlayerSettings) Marshal(io protocol.IO) {
	io.Varint32(&pk.ActionType)
}
