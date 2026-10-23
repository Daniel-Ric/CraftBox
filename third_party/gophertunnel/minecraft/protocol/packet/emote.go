package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	EmoteFlagServerSide = 1 << iota
	EmoteFlagMuteChat
)

type Emote struct {
	EntityRuntimeID uint64

	EmoteLength uint32

	EmoteID string

	XUID string

	PlatformID string

	Flags byte
}

func (*Emote) ID() uint32 {
	return IDEmote
}

func (pk *Emote) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.String(&pk.EmoteID)
	io.Varuint32(&pk.EmoteLength)
	io.String(&pk.XUID)
	io.String(&pk.PlatformID)
	io.Uint8(&pk.Flags)
}
