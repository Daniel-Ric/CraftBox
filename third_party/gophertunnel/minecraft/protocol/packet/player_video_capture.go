package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PlayerVideoCaptureActionStart = iota
	PlayerVideoCaptureActionStop
)

type PlayerVideoCapture struct {
	Action uint32

	FrameRate int32

	FilePrefix string
}

func (*PlayerVideoCapture) ID() uint32 {
	return IDPlayerVideoCapture
}

func (pk *PlayerVideoCapture) Marshal(io protocol.IO) {
	io.Varuint32(&pk.Action)
	if pk.Action == PlayerVideoCaptureActionStart {
		io.Int32(&pk.FrameRate)
		io.String(&pk.FilePrefix)
	}
}
