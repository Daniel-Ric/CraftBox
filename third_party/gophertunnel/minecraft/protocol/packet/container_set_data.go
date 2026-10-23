package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ContainerDataFurnaceTickCount = iota
	ContainerDataFurnaceLitTime
	ContainerDataFurnaceLitDuration
	_
	ContainerDataFurnaceFuelAux
)

const (
	ContainerDataBrewingStandBrewTime = iota
	ContainerDataBrewingStandFuelAmount
	ContainerDataBrewingStandFuelTotal
)

type ContainerSetData struct {
	WindowID byte

	Key int32

	Value int32
}

func (*ContainerSetData) ID() uint32 {
	return IDContainerSetData
}

func (pk *ContainerSetData) Marshal(io protocol.IO) {
	io.Uint8(&pk.WindowID)
	io.Varint32(&pk.Key)
	io.Varint32(&pk.Value)
}
