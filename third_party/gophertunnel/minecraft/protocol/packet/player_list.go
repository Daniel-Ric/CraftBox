package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PlayerList struct {
	Entries []protocol.PlayerListEntry
}

func (*PlayerList) ID() uint32 {
	return IDPlayerList
}

func (pk *PlayerList) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Entries)
}
