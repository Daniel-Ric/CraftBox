package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ShowCreditsStatusStart = iota
	ShowCreditsStatusEnd
)

type ShowCredits struct {
	PlayerRuntimeID uint64

	StatusType int32
}

func (*ShowCredits) ID() uint32 {
	return IDShowCredits
}

func (pk *ShowCredits) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.PlayerRuntimeID)
	io.Varint32(&pk.StatusType)
}
