package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ServerSettingsRequest struct {
}

func (*ServerSettingsRequest) ID() uint32 {
	return IDServerSettingsRequest
}

func (*ServerSettingsRequest) Marshal(protocol.IO) {}
