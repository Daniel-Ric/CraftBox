package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PositionTrackingDBRequestActionQuery = iota
)

type PositionTrackingDBClientRequest struct {
	RequestAction byte

	TrackingID int32
}

func (*PositionTrackingDBClientRequest) ID() uint32 {
	return IDPositionTrackingDBClientRequest
}

func (pk *PositionTrackingDBClientRequest) Marshal(io protocol.IO) {
	io.Uint8(&pk.RequestAction)
	io.Varint32(&pk.TrackingID)
}
