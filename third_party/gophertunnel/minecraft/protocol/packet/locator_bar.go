package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type LocatorBar struct {
	Waypoints []protocol.LocatorBarWaypoint
}

func (*LocatorBar) ID() uint32 {
	return IDLocatorBar
}

func (pk *LocatorBar) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Waypoints)
}
