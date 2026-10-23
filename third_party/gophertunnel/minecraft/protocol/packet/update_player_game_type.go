package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type UpdatePlayerGameType struct {
	GameType int32

	PlayerUniqueID int64

	Tick uint64
}

func (*UpdatePlayerGameType) ID() uint32 {
	return IDUpdatePlayerGameType
}

func (pk *UpdatePlayerGameType) Marshal(io protocol.IO) {
	io.Varint32(&pk.GameType)
	io.ActorUniqueID(&pk.PlayerUniqueID)
	io.Varuint64(&pk.Tick)
}
