package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type LecternUpdate struct {
	Page byte

	PageCount byte

	Position protocol.BlockPos
}

func (*LecternUpdate) ID() uint32 {
	return IDLecternUpdate
}

func (pk *LecternUpdate) Marshal(io protocol.IO) {
	io.Uint8(&pk.Page)
	io.Uint8(&pk.PageCount)
	io.BlockPos(&pk.Position)
}
