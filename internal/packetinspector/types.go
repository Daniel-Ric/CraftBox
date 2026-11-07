package packetinspector

import "encoding/json"

type Packet struct {
	Decoded            string `json:"decoded,omitempty"`
	Summary            string `json:"summary,omitempty"`
	SourceAddress      string `json:"sourceAddress,omitempty"`
	DestinationAddress string `json:"destinationAddress,omitempty"`
	ConnectionLeg      string `json:"connectionLeg,omitempty"`

	OriginSession string `json:"session,omitempty"`
	PreviewToken  string `json:"previewToken,omitempty"`
	Header        []byte `json:"header,omitempty"`
	ShieldID      int32  `json:"shieldId,omitempty"`
	ID            string `json:"id"`
	Time          int64  `json:"time"`
	PacketID      uint32 `json:"packetId"`
	Name          string `json:"name"`
	Direction     string `json:"direction"`
	Stage         string `json:"stage"`
	Bytes         int    `json:"bytes"`
	Raw           []byte `json:"raw,omitempty"`
	Payload       string `json:"payload"`
	Before        string `json:"before,omitempty"`
	Action        string `json:"action,omitempty"`
	Reason        string `json:"reason,omitempty"`
	Redacted      bool   `json:"redacted"`
	Partial       bool   `json:"partial"`
	Note          string `json:"note,omitempty"`
	Bookmarked    bool   `json:"bookmarked"`
}
type Capture struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Source    string `json:"source"`
	Protocol  int    `json:"protocol"`
	Started   int64  `json:"started"`
	Ended     int64  `json:"ended"`
	Count     uint64 `json:"count"`
	Bytes     int64  `json:"bytes"`
	DiskBytes int64  `json:"diskBytes"`
	Dropped   uint64 `json:"dropped"`
	Notes     string `json:"notes"`
	Error     string `json:"error,omitempty"`
}
type Query struct {
	Capture    string `json:"capture"`
	Search     string `json:"search"`
	Name       string `json:"name"`
	Direction  string `json:"direction"`
	Stage      string `json:"stage"`
	Action     string `json:"action"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
	Before     uint64 `json:"before"`
	After      uint64 `json:"after"`
	Bookmarked bool   `json:"bookmarked"`
	Reverse    bool   `json:"reverse"`
}
type Page struct {
	Rows    []Packet `json:"rows"`
	Total   uint64   `json:"total"`
	Capture Capture  `json:"capture"`
}
type Annotation struct {
	Note       string `json:"note"`
	Bookmarked bool   `json:"bookmarked"`
	Color      string `json:"color,omitempty"`
}
type CaptureUpdate struct {
	Capture    string     `json:"capture"`
	Name       string     `json:"name"`
	Notes      string     `json:"notes"`
	Packet     string     `json:"packet"`
	Annotation Annotation `json:"annotation"`
}
type TypeStat struct {
	Other       bool   `json:"other,omitempty"`
	Name        string `json:"name"`
	PacketID    uint32 `json:"packetId"`
	Count       uint64 `json:"count"`
	Bytes       int64  `json:"bytes"`
	Serverbound uint64 `json:"serverbound"`
	Clientbound uint64 `json:"clientbound"`
	First       int64  `json:"first"`
	Last        int64  `json:"last"`
}
type TimeBucket struct {
	Time        int64  `json:"time"`
	Serverbound uint64 `json:"serverbound"`
	Clientbound uint64 `json:"clientbound"`
	Bytes       int64  `json:"bytes"`
}
type Evidence struct {
	ID        string                     `json:"id"`
	Time      int64                      `json:"time"`
	Name      string                     `json:"name"`
	Direction string                     `json:"direction"`
	Summary   string                     `json:"summary"`
	Fields    map[string]json.RawMessage `json:"fields,omitempty"`
}
type Entity struct {
	Key       string       `json:"key"`
	Session   string       `json:"session"`
	ID        string       `json:"id"`
	Type      string       `json:"type"`
	Name      string       `json:"name"`
	Removed   bool         `json:"removed"`
	Partial   bool         `json:"partial"`
	Position  [3]float64   `json:"position"`
	Positions [][3]float64 `json:"positions"`
	Evidence  []string     `json:"evidence"`
}
type Analysis struct {
	Capture      Capture      `json:"capture"`
	Types        []TypeStat   `json:"types"`
	Buckets      []TimeBucket `json:"buckets"`
	Timeline     []Evidence   `json:"timeline"`
	Entities     []Entity     `json:"entities"`
	Transactions []Evidence   `json:"transactions"`
	Chunks       []Evidence   `json:"chunks"`
	Changes      []Evidence   `json:"changes"`
	Partial      bool         `json:"partial"`
}
type Settings struct {
	Recording     bool  `json:"recording"`
	Intervention  bool  `json:"intervention"`
	MaxDiskMB     int64 `json:"maxDiskMB"`
	CaptureWorlds bool  `json:"captureWorlds"`
	CaptureSkins  bool  `json:"captureSkins"`
	CapturePacks  bool  `json:"capturePacks"`
}
type Status struct {
	RecordingIDs []string  `json:"recordingIDs"`
	Settings     Settings  `json:"settings"`
	Captures     []Capture `json:"captures"`
	Active       string    `json:"active"`
	Connected    bool      `json:"connected"`
	Generation   string    `json:"generation"`
	QueueDropped uint64    `json:"queueDropped"`
	Error        string    `json:"error"`
	Rules        []Rule    `json:"rules"`
}
