package protocol

const (
	ClockPayloadTypeSyncState = iota
	ClockPayloadTypeInitializeRegistry
	ClockPayloadTypeAddTimeMarker
	ClockPayloadTypeRemoveTimeMarker
)

type SyncWorldClockStateData struct {
	ClockID uint64

	Time int32

	Paused bool
}

func (x *SyncWorldClockStateData) Marshal(r IO) {
	r.Varuint64(&x.ClockID)
	r.Varint32(&x.Time)
	r.Bool(&x.Paused)
}

type TimeMarkerData struct {
	ID uint64

	Name string

	Time int32

	Period Optional[int32]
}

func (x *TimeMarkerData) Marshal(r IO) {
	r.Varuint64(&x.ID)
	r.String(&x.Name)
	r.Varint32(&x.Time)
	OptionalFunc(r, &x.Period, r.Int32)
}

type WorldClockData struct {
	ID uint64

	Name string

	Time int32

	Paused bool

	TimeMarkers []TimeMarkerData
}

func (x *WorldClockData) Marshal(r IO) {
	r.Varuint64(&x.ID)
	r.String(&x.Name)
	r.Varint32(&x.Time)
	r.Bool(&x.Paused)
	Slice(r, &x.TimeMarkers)
}
