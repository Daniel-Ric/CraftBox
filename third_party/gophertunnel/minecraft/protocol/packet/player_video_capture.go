package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PlayerVideoCaptureActionStop = iota
	PlayerVideoCaptureActionStart
)

type PlayerVideoCapture struct {
	Action byte

	FrameRate int32

	FilePrefix string
}

func (*PlayerVideoCapture) ID() uint32 {
	return IDPlayerVideoCapture
}

func (pk *PlayerVideoCapture) Marshal(io protocol.IO) {
	io.Uint8(&pk.Action)
	if pk.Action == PlayerVideoCaptureActionStart {
		io.Int32(&pk.FrameRate)
		io.String(&pk.FilePrefix)
	}
}
