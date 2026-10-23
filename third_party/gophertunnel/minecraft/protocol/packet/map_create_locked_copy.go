package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type MapCreateLockedCopy struct {
	OriginalMapID int64

	NewMapID int64
}

func (*MapCreateLockedCopy) ID() uint32 {
	return IDMapCreateLockedCopy
}

func (pk *MapCreateLockedCopy) Marshal(io protocol.IO) {
	io.Varint64(&pk.OriginalMapID)
	io.Varint64(&pk.NewMapID)
}
