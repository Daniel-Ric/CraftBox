package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ServerPresenceInfo struct {
	PresenceInfo protocol.Optional[protocol.PresenceInfo]
}

func (*ServerPresenceInfo) ID() uint32 {
	return IDServerPresenceInfo
}

func (pk *ServerPresenceInfo) Marshal(io protocol.IO) {
	protocol.OptionalMarshaler(io, &pk.PresenceInfo)
}
