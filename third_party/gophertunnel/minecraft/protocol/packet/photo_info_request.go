package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PhotoInfoRequest struct {
	PhotoID int64
}

func (*PhotoInfoRequest) ID() uint32 {
	return IDPhotoInfoRequest
}

func (pk *PhotoInfoRequest) Marshal(io protocol.IO) {
	io.Varint64(&pk.PhotoID)
}
