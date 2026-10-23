package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ContainerClose struct {
	WindowID byte

	ContainerType byte

	ServerSide bool
}

func (*ContainerClose) ID() uint32 {
	return IDContainerClose
}

func (pk *ContainerClose) Marshal(io protocol.IO) {
	io.Uint8(&pk.WindowID)
	io.Uint8(&pk.ContainerType)
	io.Bool(&pk.ServerSide)
}
