package packet

import (
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ServerBoundPackSettingChange struct {
	PackID uuid.UUID

	PackSetting protocol.PackSetting
}

func (*ServerBoundPackSettingChange) ID() uint32 {
	return IDServerBoundPackSettingChange
}

func (pk *ServerBoundPackSettingChange) Marshal(io protocol.IO) {
	io.UUID(&pk.PackID)
	io.PackSetting(&pk.PackSetting)
}
