package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetDefaultGameType struct {
	GameType int32
}

func (*SetDefaultGameType) ID() uint32 {
	return IDSetDefaultGameType
}

func (pk *SetDefaultGameType) Marshal(io protocol.IO) {
	io.Varint32(&pk.GameType)
}
