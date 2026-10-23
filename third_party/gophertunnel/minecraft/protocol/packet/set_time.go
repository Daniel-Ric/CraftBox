package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetTime struct {
	Time int32
}

func (*SetTime) ID() uint32 {
	return IDSetTime
}

func (pk *SetTime) Marshal(io protocol.IO) {
	io.Varint32(&pk.Time)
}
