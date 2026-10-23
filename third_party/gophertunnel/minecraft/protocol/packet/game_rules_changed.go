package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type GameRulesChanged struct {
	GameRules []protocol.GameRule
}

func (*GameRulesChanged) ID() uint32 {
	return IDGameRulesChanged
}

func (pk *GameRulesChanged) Marshal(io protocol.IO) {
	protocol.FuncSlice(io, &pk.GameRules, io.GameRule)
}
