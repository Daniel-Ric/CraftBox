package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ContainerOpen struct {
	WindowID byte

	ContainerType byte

	ContainerPosition protocol.BlockPos

	ContainerEntityUniqueID int64
}

func (*ContainerOpen) ID() uint32 {
	return IDContainerOpen
}

func (pk *ContainerOpen) Marshal(io protocol.IO) {
	io.Uint8(&pk.WindowID)
	io.Uint8(&pk.ContainerType)
	io.BlockPos(&pk.ContainerPosition)
	io.ActorUniqueID(&pk.ContainerEntityUniqueID)
}
