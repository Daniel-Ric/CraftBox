package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	DataDrivenScreenCloseReasonProgrammaticClose    = "programmaticclose"
	DataDrivenScreenCloseReasonProgrammaticCloseAll = "programmaticcloseall"
	DataDrivenScreenCloseReasonClientCanceled       = "clientcanceled"
	DataDrivenScreenCloseReasonUserBusy             = "userbusy"
	DataDrivenScreenCloseReasonInvalidForm          = "invalidform"
)

type ServerBoundDataDrivenScreenClosed struct {
	FormID uint32

	CloseReason string
}

func (*ServerBoundDataDrivenScreenClosed) ID() uint32 {
	return IDServerBoundDataDrivenScreenClosed
}

func (pk *ServerBoundDataDrivenScreenClosed) Marshal(io protocol.IO) {
	io.Uint32(&pk.FormID)
	io.String(&pk.CloseReason)
}
