package packetinspector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Sender func(packet.Packet, bool) error
type queuedPacket struct {
	moment  *captureMoment
	record  bool
	capture string
	packet  Packet
	cost    int64
}
type Controller struct {
	triggerMu                     sync.Mutex
	history                       []triggerHistory
	historyBytes                  int64
	windows                       []triggerWindow
	windowCount                   atomic.Int32
	triggerCooldown               map[string]int64
	Store                         *Store
	Engine                        *Engine
	mu                            sync.Mutex
	settings                      Settings
	active, generation, lastError string
	failedArchive                 string
	sender                        Sender
	queue                         chan queuedPacket
	done                          chan struct{}
	queued                        atomic.Int64
	dropped                       atomic.Uint64
	closed                        bool
	losses                        map[string]uint64
	previews                      map[string]sendPreview
	finalizing                    map[string]bool
	deleting                      map[string]bool
}

func NewController(root string) *Controller {
	c := &Controller{Store: New(root), Engine: NewEngine(), settings: Settings{Recording: true, MaxDiskMB: 4096, CapturePacks: true, CaptureSkins: true}, queue: make(chan queuedPacket, 1024), done: make(chan struct{}), losses: map[string]uint64{}}
	c.previews = map[string]sendPreview{}
	c.finalizing = map[string]bool{}
	c.deleting = map[string]bool{}
	c.triggerCooldown = map[string]int64{}
	c.loadPreferences()
	go c.worker()
	return c
}
func (c *Controller) worker() {
	defer close(c.done)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.expireTriggers()
			c.persistGaps()
		case q, ok := <-c.queue:
			if !ok {
				c.finishTriggers("")
				return
			}
			c.processPacket(q)
		}
	}
}
func (c *Controller) processPacket(q queuedPacket) {
	if q.moment != nil {
		defer c.queued.Add(-q.cost)
		page, err := c.Store.Query(Query{Capture: q.capture, Reverse: true, Limit: 400})
		if err != nil {
			c.Report("Capture marker: " + err.Error())
			return
		}
		for _, p := range page.Rows {
			if p.Time <= q.moment.at {
				note := p.Note
				if note != "" {
					note += "\n"
				}
				note += q.moment.note
				err = c.Store.Update(CaptureUpdate{Capture: q.capture, Packet: p.ID, Annotation: Annotation{Bookmarked: true, Note: note, Color: "#e7b66d"}})
				if err != nil {
					c.Report("Capture marker: " + err.Error())
				}
				return
			}
		}
		return
	}
	if q.packet.Payload == "" {
		source := q.packet
		q.packet = FromRaw(source.PacketID, source.Raw, source.Direction, source.Stage, time.UnixMilli(source.Time), source.ShieldID)
		q.packet.Header = source.Header
		q.packet.SourceAddress = source.SourceAddress
		q.packet.DestinationAddress = source.DestinationAddress
		q.packet.ConnectionLeg = source.ConnectionLeg
	}
	p := q.packet
	p.OriginSession = q.capture
	decision := Decision{Action: "forward"}
	if p.Stage == "received" {
		decision = c.Engine.Observe(p)
		if decision.Action != "forward" {
			p.Bookmarked = true
			p.Note = "Watchpoint " + decision.Rule
		}
	}
	c.triggerPacket(q, p, decision)
	var e error
	if q.record {
		c.mu.Lock()
		failed := c.failedArchive == q.capture
		c.mu.Unlock()
		if failed {
			c.mu.Lock()
			c.losses[q.capture]++
			c.mu.Unlock()
			q.record = false
		} else {
			p, e = c.Store.Append(q.capture, p)
		}
	}
	if e != nil {
		c.mu.Lock()
		c.lastError = e.Error()
		c.settings.Recording = false
		c.failedArchive = q.capture
		c.mu.Unlock()
	} else if q.record && (p.Bookmarked || p.Note != "") {
		_ = c.Store.Update(CaptureUpdate{Capture: q.capture, Packet: p.ID, Annotation: Annotation{Note: p.Note, Bookmarked: p.Bookmarked}})
	}
	c.queued.Add(-q.cost)
	if c.queued.Load() == 0 {
		c.persistGaps()
	}
}
func (c *Controller) persistGaps() {
	c.mu.Lock()
	if len(c.losses) == 0 {
		c.mu.Unlock()
		return
	}
	losses := c.losses
	c.losses = map[string]uint64{}
	c.mu.Unlock()
	for id, count := range losses {
		c.Store.Gap(id, count, "Recorder queue overflow; capture is incomplete")
	}
}
func (c *Controller) Configure(s Settings) error {
	if s.MaxDiskMB < 16 || s.MaxDiskMB > 1024*1024 {
		return errors.New("Archive limit must be between 16 MiB and 1 TiB")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.settings = s
	if s.Recording {
		c.failedArchive = ""
	}
	c.lastError = ""
	c.Engine.Enable(s.Intervention)
	c.Store.mu.Lock()
	c.Store.MaxBytes = s.MaxDiskMB << 20
	c.Store.mu.Unlock()
	return c.savePreferences()
}
func (c *Controller) Settings() Settings    { c.mu.Lock(); defer c.mu.Unlock(); return c.settings }
func (c *Controller) Report(message string) { c.mu.Lock(); c.lastError = message; c.mu.Unlock() }
func (c *Controller) Begin(name string, protocol int, send Sender) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return "", errors.New("Inspector is closed")
	}
	if c.sender != nil {
		return "", errors.New("Only one active NetherNet session is supported")
	}
	if len(c.finalizing) > 0 {
		return "", errors.New("The previous capture is still being finalized")
	}
	a, e := c.Store.Create(name, "NetherNet", protocol)
	if e != nil {
		return "", e
	}
	c.active = a.ID
	c.failedArchive = ""
	c.generation = a.ID
	c.sender = send
	c.lastError = ""
	return a.ID, nil
}
func (c *Controller) End(id string) {
	if id == "" {
		return
	}
	c.mu.Lock()
	if id != "" {
		c.finalizing[id] = true
	}
	if c.active == id {
		c.sender = nil
		c.settings.Intervention = false
		c.Engine.Enable(false)
	}
	c.mu.Unlock()
	c.Flush()
	c.finishTriggers(id)
	if e := c.Store.Finish(id); e != nil {
		c.Report("Capture finalization failed: " + e.Error())
	}
	c.mu.Lock()
	delete(c.finalizing, id)
	c.mu.Unlock()
}
func (c *Controller) Delete(id string) error {
	c.triggerMu.Lock()
	for _, window := range c.windows {
		if window.capture == id {
			c.triggerMu.Unlock()
			return errors.New("Wait for the triggered recording to finish before deleting")
		}
	}
	c.mu.Lock()
	if c.finalizing[id] || c.sender != nil && c.active == id {
		c.mu.Unlock()
		c.triggerMu.Unlock()
		return errors.New("Wait for the NetherNet session and capture finalization to finish before deleting")
	}
	if c.deleting[id] {
		c.mu.Unlock()
		c.triggerMu.Unlock()
		return errors.New("This recording is already being deleted")
	}
	c.deleting[id] = true
	c.mu.Unlock()
	c.triggerMu.Unlock()
	// Waiting for snapshot handles must never hold the controller locks used
	// by an unrelated live session's observation, status or trigger handling.
	defer func() {
		c.mu.Lock()
		delete(c.deleting, id)
		c.mu.Unlock()
	}()
	return c.Store.Delete(id)
}
func (c *Controller) Observe(id string, p Packet) {
	if p.Time == 0 {
		p.Time = time.Now().UnixMilli()
	}
	cost := int64(len(p.Raw) + len(p.Payload) + len(p.Before) + 512)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.deleting[id] || id == "" || !c.settings.Recording && !(p.Stage == "received" && (c.windowCount.Load() > 0 || c.Engine.HasTriggers())) {
		return
	}
	if cost > maxRecord || c.queued.Load()+cost > 64<<20 {
		c.dropped.Add(1)
		c.losses[id]++
		c.lastError = "Recorder queue limit reached; observations were dropped"
		return
	}
	p.Raw = append([]byte{}, p.Raw...)
	c.queued.Add(cost)
	select {
	case c.queue <- queuedPacket{record: c.settings.Recording, capture: id, packet: p, cost: cost}:
	default:
		c.queued.Add(-cost)
		c.dropped.Add(1)
		c.losses[id]++
		c.lastError = "Recorder queue is full; observations were dropped"
	}
}
func (c *Controller) Flush() {
	for c.queued.Load() > 0 {
		time.Sleep(time.Millisecond)
	}
	c.persistGaps()
}

type captureMoment struct {
	at   int64
	note string
}

// Queue behind received packets so a command can mark its recorded moment
// without racing the asynchronous archive writer or generating a fake packet.
func (c *Controller) MarkMoment(at time.Time, note string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.active == "" || c.sender == nil || !c.settings.Recording {
		return
	}
	c.queued.Add(1)
	select {
	case c.queue <- queuedPacket{capture: c.active, cost: 1, moment: &captureMoment{at.UnixMilli(), note}}:
	default:
		c.queued.Add(-1)
		c.lastError = "Capture marker could not be linked: recorder queue is full"
	}
}
func (c *Controller) Status() (Status, error) {
	c.triggerMu.Lock()
	recordings := []string{}
	for _, window := range c.windows {
		recordings = append(recordings, window.capture)
	}
	c.triggerMu.Unlock()
	c.mu.Lock()
	out := Status{Settings: c.settings, Active: c.active, Connected: c.sender != nil, Generation: c.generation, QueueDropped: c.dropped.Load(), Error: c.lastError, Rules: c.Engine.Summaries()}
	c.mu.Unlock()
	out.RecordingIDs = recordings
	var e error
	out.Captures, e = c.Store.List()
	return out, e
}
func (c *Controller) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.sender = nil
	close(c.queue)
	c.mu.Unlock()
	<-c.done
	c.persistGaps()
	c.Store.Close()
}
func PacketName(pk packet.Packet) string {
	t := reflect.TypeOf(pk)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}

var serverPackets = minecraft.DefaultProtocol.Packets(false)
var clientPackets = minecraft.DefaultProtocol.Packets(true)

func currentProtocol() int { return protocol.CurrentProtocol }

func sensitivePacketID(id uint32) bool {
	f := serverPackets[id]
	if f == nil {
		f = clientPackets[id]
	}
	return f != nil && sensitiveName(PacketName(f()))
}
func Decode(id uint32, raw []byte, shield int32) (pk packet.Packet, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Decode %d: %v", id, r)
			pk = nil
		}
	}()
	pool := serverPackets
	f := pool[id]
	if f == nil {
		f = clientPackets[id]
	}
	if f == nil {
		return &packet.Unknown{PacketID: id, Payload: append([]byte{}, raw...)}, nil
	}
	pk = f()
	pk.Marshal(protocol.NewReader(bytes.NewBuffer(raw), shield, false))
	return pk, nil
}
func Encode(pk packet.Packet, shield int32) (raw []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Encode packet: %v", r)
		}
	}()
	b := bytes.NewBuffer(nil)
	pk.Marshal(protocol.NewWriter(b, shield))
	return b.Bytes(), nil
}
func DecodeJSON(id uint32, text string) (pk packet.Packet, err error) {
	if len(text) > 1<<20 || !json.Valid([]byte(text)) {
		return nil, errors.New("Packet JSON must be valid and at most 1 MiB")
	}
	f := serverPackets[id]
	if f == nil {
		f = clientPackets[id]
	}
	if f == nil {
		return nil, errors.New("Unknown packet type cannot be edited as JSON; use raw bytes")
	}
	pk = f()
	d := json.NewDecoder(strings.NewReader(text))
	d.DisallowUnknownFields()
	if e := d.Decode(pk); e != nil {
		return nil, e
	}
	if pk.ID() != id {
		return nil, errors.New("Packet type changed")
	}
	return pk, nil
}
func FromRaw(id uint32, raw []byte, direction, stage string, at time.Time, shield int32) Packet {
	p := Packet{ShieldID: shield, Time: at.UnixMilli(), PacketID: id, Name: "Unknown " + strconv.FormatUint(uint64(id), 10), Direction: direction, Stage: stage, Bytes: len(raw), Raw: raw, Payload: "{}"}
	if f := serverPackets[id]; f != nil {
		p.Name = PacketName(f())
	} else if f := clientPackets[id]; f != nil {
		p.Name = PacketName(f())
	}
	pk, e := Decode(id, raw, shield)
	if e != nil {
		p.Partial = true
		p.Reason = e.Error()
		return p
	}
	p.Name = PacketName(pk)
	if _, unknown := pk.(*packet.Unknown); unknown {
		p.Partial = true
		p.Payload = `{"undecoded":true}`
		return p
	}
	body, e := json.Marshal(pk)
	if e != nil {
		p.Partial = true
		p.Reason = e.Error()
	} else {
		p.Payload = string(body)
		p.Decoded = inspectPayload(pk)
	}
	return p
}
func (c *Controller) Intercept(ctx context.Context, id string, pk packet.Packet, toServer bool) (packet.Packet, error) {
	if id == "" {
		return pk, nil
	}
	direction := "clientbound"
	if toServer {
		direction = "serverbound"
	}
	if !c.Engine.Interested(PacketName(pk), direction) {
		return pk, nil
	}
	body, e := json.Marshal(pk)
	if e != nil {
		return pk, nil
	}
	p := Packet{Time: time.Now().UnixMilli(), PacketID: pk.ID(), Name: PacketName(pk), Direction: direction, Stage: "decision", Payload: string(body)}
	d := c.Engine.Decide(p)
	if d.Action == "forward" {
		return pk, nil
	}
	p.Action = d.Action
	p.Reason = "Rule " + d.Rule
	switch d.Action {
	case "block":
		c.Observe(id, p)
		return nil, nil
	case "edit":
		replacement, e := DecodeJSON(pk.ID(), d.Payload)
		if e == nil {
			_, e = Encode(replacement, 0)
		}
		if e != nil {
			p.Action = "rejected"
			p.Reason = e.Error()
			c.Observe(id, p)
			return pk, nil
		}
		p.Before = p.Payload
		p.Payload = d.Payload
		c.Observe(id, p)
		return replacement, nil
	case "delay":
		timer := time.NewTimer(time.Duration(d.DelayMS) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
		c.Observe(id, p)
	case "bookmark", "trigger":
		p.Bookmarked = true
		p.Note = "Watchpoint " + d.Rule
		c.Observe(id, p)
	}
	return pk, nil
}

type SendRequest struct {
	PreviewToken string `json:"previewToken"`
	Generation   string `json:"generation"`
	Direction    string `json:"direction"`
	PacketID     uint32 `json:"packetId"`
	Payload      string `json:"payload"`
	Raw          []byte `json:"raw"`
	UseRaw       bool   `json:"useRaw"`
}

func PrepareSend(r SendRequest, shield int32) (packet.Packet, Packet, error) {
	if r.PacketID > 1023 {
		return nil, Packet{}, errors.New("Packet ID must fit the Bedrock packet header (0–1023)")
	}
	direction := r.Direction
	if direction != "clientbound" && direction != "serverbound" {
		return nil, Packet{}, errors.New("Choose a send direction")
	}
	if len(r.Raw) > 1<<20 {
		return nil, Packet{}, errors.New("Send up to 1 MiB at a time")
	}
	var pk packet.Packet
	var e error
	if r.UseRaw {
		pk = &packet.Unknown{PacketID: r.PacketID, Payload: append([]byte{}, r.Raw...)}
	} else {
		pk, e = DecodeJSON(r.PacketID, r.Payload)
	}
	if e != nil {
		return nil, Packet{}, e
	}
	raw, e := Encode(pk, shield)
	if e != nil {
		return nil, Packet{}, e
	}
	p := FromRaw(r.PacketID, raw, direction, "injected", time.Now(), shield)
	p.Action = "send"
	return pk, p, nil
}
func (c *Controller) Send(r SendRequest, shield int32) (Packet, error) {
	pk, p, e := PrepareSend(r, shield)
	if e != nil {
		return p, e
	}
	c.mu.Lock()
	if !c.settings.Intervention || c.sender == nil || c.generation != r.Generation || r.Generation == "" {
		c.mu.Unlock()
		return p, errors.New("Enable intervention and preview against the current active session")
	}
	if e = c.consumePreview(r, p); e != nil {
		c.mu.Unlock()
		return p, e
	}
	sender, id := c.sender, c.active
	c.mu.Unlock()
	if e = sender(pk, r.Direction == "serverbound"); e != nil {
		return p, e
	}
	c.Observe(id, p)
	return p, nil
}
