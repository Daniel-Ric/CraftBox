package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type MapInfoRequest struct {
	MapID int64

	ClientPixels []protocol.PixelRequest
}

func (*MapInfoRequest) ID() uint32 {
	return IDMapInfoRequest
}

func (pk *MapInfoRequest) Marshal(io protocol.IO) {
	io.Varint64(&pk.MapID)
	protocol.SliceUint32Length(io, &pk.ClientPixels)
}
