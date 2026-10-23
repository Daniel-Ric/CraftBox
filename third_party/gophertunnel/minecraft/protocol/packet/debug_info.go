package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type DebugInfo struct {
	PlayerUniqueID int64

	Data []byte
}

func (*DebugInfo) ID() uint32 {
	return IDDebugInfo
}

func (pk *DebugInfo) Marshal(io protocol.IO) {
	io.ActorUniqueID(&pk.PlayerUniqueID)
	io.ByteSlice(&pk.Data)
}
