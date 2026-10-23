package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SyncWorldClocks struct {
	PayloadType uint32

	SyncStates []protocol.SyncWorldClockStateData

	Clocks []protocol.WorldClockData

	AddClockID uint64

	AddTimeMarkers []protocol.TimeMarkerData

	RemoveClockID uint64

	RemoveTimeMarkerIDs []uint64
}

func (*SyncWorldClocks) ID() uint32 {
	return IDSyncWorldClocks
}

func (pk *SyncWorldClocks) Marshal(io protocol.IO) {
	io.Varuint32(&pk.PayloadType)
	switch pk.PayloadType {
	case protocol.ClockPayloadTypeSyncState:
		protocol.Slice(io, &pk.SyncStates)
	case protocol.ClockPayloadTypeInitializeRegistry:
		protocol.Slice(io, &pk.Clocks)
	case protocol.ClockPayloadTypeAddTimeMarker:
		io.Varuint64(&pk.AddClockID)
		protocol.Slice(io, &pk.AddTimeMarkers)
	case protocol.ClockPayloadTypeRemoveTimeMarker:
		io.Varuint64(&pk.RemoveClockID)
		protocol.FuncSlice(io, &pk.RemoveTimeMarkerIDs, io.Varuint64)
	default:
		io.UnknownEnumOption(pk.PayloadType, "clock payload type")
	}
}
