package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ScoreboardIdentityActionRegister = iota
	ScoreboardIdentityActionClear
)

type SetScoreboardIdentity struct {
	ActionType byte

	Entries []protocol.ScoreboardIdentityEntry
}

func (*SetScoreboardIdentity) ID() uint32 {
	return IDSetScoreboardIdentity
}

func (pk *SetScoreboardIdentity) Marshal(io protocol.IO) {
	io.Uint8(&pk.ActionType)
	if pk.ActionType != ScoreboardIdentityActionRegister && pk.ActionType != ScoreboardIdentityActionClear {
		io.UnknownEnumOption(pk.ActionType, "scoreboard identity action type")
	}
	protocol.Slice(io, &pk.Entries)
}
