package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ModalFormCancelReasonUserClosed = iota
	ModalFormCancelReasonUserBusy
)

type ModalFormResponse struct {
	FormID uint32

	ResponseData protocol.Optional[[]byte]

	CancelReason protocol.Optional[uint8]
}

func (*ModalFormResponse) ID() uint32 {
	return IDModalFormResponse
}

func (pk *ModalFormResponse) Marshal(io protocol.IO) {
	io.Varuint32(&pk.FormID)
	protocol.OptionalFunc(io, &pk.ResponseData, io.ByteSlice)
	protocol.OptionalFunc(io, &pk.CancelReason, io.Uint8)
}
