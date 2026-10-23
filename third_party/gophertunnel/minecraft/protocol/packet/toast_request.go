package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type ToastRequest struct {
	Title string

	Message string
}

func (*ToastRequest) ID() uint32 {
	return IDToastRequest
}

func (pk *ToastRequest) Marshal(io protocol.IO) {
	io.String(&pk.Title)
	io.String(&pk.Message)
}
