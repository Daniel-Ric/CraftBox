package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type LevelEventGeneric struct {
	EventID int32

	SerialisedEventData []byte
}

func (pk *LevelEventGeneric) ID() uint32 {
	return IDLevelEventGeneric
}

func (pk *LevelEventGeneric) Marshal(io protocol.IO) {
	io.Varint32(&pk.EventID)
	io.Bytes(&pk.SerialisedEventData)
}
