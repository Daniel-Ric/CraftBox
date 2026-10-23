package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ServerSettingsResponse struct {
	FormID uint32

	FormData []byte
}

func (*ServerSettingsResponse) ID() uint32 {
	return IDServerSettingsResponse
}

func (pk *ServerSettingsResponse) Marshal(io protocol.IO) {
	io.Varuint32(&pk.FormID)
	io.ByteSlice(&pk.FormData)
}
