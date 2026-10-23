package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ModalFormRequest struct {
	FormID uint32

	FormData []byte
}

func (*ModalFormRequest) ID() uint32 {
	return IDModalFormRequest
}

func (pk *ModalFormRequest) Marshal(io protocol.IO) {
	io.Varuint32(&pk.FormID)
	io.ByteSlice(&pk.FormData)
}
