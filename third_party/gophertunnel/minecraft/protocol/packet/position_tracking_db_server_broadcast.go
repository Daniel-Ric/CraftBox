package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PositionTrackingDBBroadcastActionUpdate = iota
	PositionTrackingDBBroadcastActionDestroy
	PositionTrackingDBBroadcastActionNotFound
)

type PositionTrackingDBServerBroadcast struct {
	BroadcastAction byte

	TrackingID int32

	Payload map[string]any
}

func (*PositionTrackingDBServerBroadcast) ID() uint32 {
	return IDPositionTrackingDBServerBroadcast
}

func (pk *PositionTrackingDBServerBroadcast) Marshal(io protocol.IO) {
	io.Uint8(&pk.BroadcastAction)
	io.Varint32(&pk.TrackingID)
	io.NBT(&pk.Payload, nbt.NetworkLittleEndian)
}
