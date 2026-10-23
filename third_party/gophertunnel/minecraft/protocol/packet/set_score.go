package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetScore struct {
	Entries []protocol.ScoreboardEntry
}

func (*SetScore) ID() uint32 {
	return IDSetScore
}

func (pk *SetScore) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Entries)
}
