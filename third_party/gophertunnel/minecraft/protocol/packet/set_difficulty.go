package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SetDifficulty struct {
	Difficulty uint32
}

func (*SetDifficulty) ID() uint32 {
	return IDSetDifficulty
}

func (pk *SetDifficulty) Marshal(io protocol.IO) {
	io.Varuint32(&pk.Difficulty)
}
