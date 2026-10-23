package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type TickingAreasLoadStatus struct {
	Preload bool
}

func (*TickingAreasLoadStatus) ID() uint32 {
	return IDTickingAreasLoadStatus
}

func (pk *TickingAreasLoadStatus) Marshal(io protocol.IO) {
	io.Bool(&pk.Preload)
}
