package packet

import (
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type EmoteList struct {
	PlayerRuntimeID uint64

	EmotePieces []uuid.UUID
}

func (*EmoteList) ID() uint32 {
	return IDEmoteList
}

func (pk *EmoteList) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.PlayerRuntimeID)
	protocol.FuncSlice(io, &pk.EmotePieces, io.UUID)
}
