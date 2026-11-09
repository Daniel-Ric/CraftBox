package commandexplorer

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Store struct {
	Root        string
	mu          sync.Mutex
	live        map[string]*Document
	cache       map[string]*Document
	queueMu     sync.Mutex
	queue       chan queuedEvent
	pending     sync.WaitGroup
	queuedBytes int
	closed      bool
}

type queuedEvent struct {
	id    string
	event Event
	cost  int
}

func New(root string) *Store {
	return &Store{Root: root, live: map[string]*Document{}, cache: map[string]*Document{}}
}
func valid(id string) bool { u, e := uuid.Parse(id); return e == nil && u.String() == id }
func writeJSON(name string, v any) error {
	raw, e := json.Marshal(v)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(name), "explorer-*.tmp")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(raw)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(f.Name(), name)
}
func (s *Store) Begin(name, capture string, version int) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()
	m := Session{ID: uuid.NewString(), Name: name, Capture: capture, Protocol: version, Started: now, Updated: now}
	if e := os.MkdirAll(filepath.Join(s.Root, m.ID), 0700); e != nil {
		return m, e
	}
	if e := writeJSON(filepath.Join(s.Root, m.ID, "session.json"), m); e != nil {
		return m, e
	}
	s.live[m.ID] = &Document{Format: "craftbox-command-archive", Version: 1, Session: m, Commands: []Command{}, Events: []Event{}, Warnings: []string{}}
	return m, nil
}
func (s *Store) List() ([]Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Session{}
	entries, e := os.ReadDir(s.Root)
	if os.IsNotExist(e) {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	for _, entry := range entries {
		if !entry.IsDir() || !valid(entry.Name()) {
			continue
		}
		var m Session
		if d := s.live[entry.Name()]; d != nil {
			m = d.Session
		} else {
			f, e := os.Open(filepath.Join(s.Root, entry.Name(), "session.json"))
			if e != nil {
				continue
			}
			e = json.NewDecoder(io.LimitReader(f, 1<<20)).Decode(&m)
			f.Close()
			if e != nil || m.ID != entry.Name() {
				continue
			}
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started > out[j].Started })
	return out, nil
}
func (s *Store) load(id string) (*Document, error) {
	if !valid(id) {
		return nil, errors.New("Invalid server session identifier")
	}
	if d := s.live[id]; d != nil {
		return d, nil
	}
	if d := s.cache[id]; d != nil {
		return d, nil
	}
	dir := filepath.Join(s.Root, id)
	info, e := os.Lstat(dir)
	if e != nil {
		return nil, e
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("Invalid server session directory")
	}
	f, e := os.Open(filepath.Join(dir, "session.json"))
	if e != nil {
		return nil, e
	}
	d := &Document{Format: "craftbox-command-archive", Version: 1, Commands: []Command{}, Events: []Event{}, Warnings: []string{}}
	e = json.NewDecoder(io.LimitReader(f, 1<<20)).Decode(&d.Session)
	f.Close()
	if e != nil {
		return nil, e
	}
	if d.Session.ID != id {
		return nil, errors.New("Invalid server session metadata")
	}
	f, e = os.Open(filepath.Join(dir, "events.jsonl"))
	if os.IsNotExist(e) {
		return d, nil
	}
	if e != nil {
		return nil, e
	}
	defer f.Close()
	info, e = f.Stat()
	if e != nil {
		return nil, e
	}
	if info.Size() > 128<<20 {
		return nil, errors.New("Server session exceeds 128 MiB")
	}
	d.Session.Revision = 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), 32<<20)
	for scanner.Scan() {
		var event Event
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			d.Warnings = append(d.Warnings, "An incomplete archive record was skipped")
			continue
		}
		if dialogEvent(event.Kind) {
			continue
		}
		if err := validateEvent(&event); err != nil {
			if len(d.Warnings) < 10 {
				d.Warnings = append(d.Warnings, err.Error())
			}
			continue
		}
		if len(d.Events) >= 30000 {
			d.Warnings = append(d.Warnings, "Archive event limit reached")
			break
		}
		apply(d, event)
	}
	if e = scanner.Err(); e != nil {
		return nil, e
	}
	if len(s.cache) >= 4 {
		for key := range s.cache {
			delete(s.cache, key)
			break
		}
	}
	s.cache[id] = d
	return d, nil
}
func (s *Store) Get(id string) (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, e := s.load(id)
	if e != nil {
		return Document{}, e
	}
	raw, e := json.Marshal(d)
	if e != nil {
		return Document{}, e
	}
	var out Document
	e = json.Unmarshal(raw, &out)
	return out, e
}
func (s *Store) Observe(id string, pk packet.Packet, toServer bool, at time.Time) error {
	return s.ObserveEvidence(id, pk, toServer, at, "")
}
func (s *Store) ObserveEvidence(id string, pk packet.Packet, toServer bool, at time.Time, sourcePacket string) error {
	event, e := prepare(pk, toServer, at, sourcePacket)
	if e != nil || event.Kind == "" {
		return e
	}
	return s.appendEvent(id, event)
}
func prepare(pk packet.Packet, toServer bool, at time.Time, sourcePacket string) (Event, error) {
	kind, data, summary, _ := normalize(pk, toServer)
	if kind == "" {
		return Event{}, nil
	}
	raw, e := json.Marshal(data)
	if e != nil {
		return Event{}, e
	}
	if len(raw) > 16<<20 {
		return Event{}, errors.New("Command Explorer event exceeds 16 MiB")
	}
	direction := "clientbound"
	if toServer {
		direction = "serverbound"
	}
	name := strings.TrimPrefix(fmt.Sprintf("%T", pk), "*packet.")
	return Event{Time: at.UnixMilli(), Kind: kind, Name: name, Direction: direction, PacketID: pk.ID(), Packet: sourcePacket, Summary: summary, Data: raw}, nil
}
func (s *Store) Record(id string, pk packet.Packet, toServer bool, at time.Time) error {
	event, e := prepare(pk, toServer, at, "")
	if e != nil || event.Kind == "" {
		return e
	}
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	if s.closed {
		return errors.New("Command Explorer recorder is closed")
	}
	if s.queue == nil {
		s.queue = make(chan queuedEvent, 256)
		go func() {
			for item := range s.queue {
				err := s.appendEvent(item.id, item.event)
				if err != nil {
					s.mu.Lock()
					if d := s.live[item.id]; d != nil {
						d.Session.Error = err.Error()
					}
					s.mu.Unlock()
				}
				s.queueMu.Lock()
				s.queuedBytes -= item.cost
				s.queueMu.Unlock()
				s.pending.Done()
			}
		}()
	}
	envelope, e := json.Marshal(event)
	if e != nil {
		return e
	}
	cost := len(envelope) + 32
	if s.queuedBytes+cost > 32<<20 {
		return errors.New("Command Explorer recorder queue exceeded 32 MiB")
	}
	s.pending.Add(1)
	select {
	case s.queue <- queuedEvent{id: id, event: event, cost: cost}:
		s.queuedBytes += cost
		return nil
	default:
		s.pending.Done()
		return errors.New("Command Explorer recorder queue is full")
	}
}
func (s *Store) Close() {
	s.queueMu.Lock()
	if s.closed {
		s.queueMu.Unlock()
		return
	}
	s.closed = true
	queue := s.queue
	s.queueMu.Unlock()
	s.pending.Wait()
	if queue != nil {
		close(queue)
	}
}
func (s *Store) appendEvent(id string, event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, e := s.load(id)
	if e != nil {
		return e
	}
	if len(d.Events) >= 30000 {
		return errors.New("Command Explorer session limit reached")
	}
	event.ID = strconv.Itoa(len(d.Events) + 1)
	line, e := json.Marshal(event)
	if e != nil {
		return e
	}
	if d.Session.Bytes+int64(len(line)+1) > 128<<20 {
		return errors.New("Command Explorer session limit reached")
	}
	f, e := os.OpenFile(filepath.Join(s.Root, id, "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(append(line, '\n'))
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	apply(d, event)
	d.Session.Bytes += int64(len(line) + 1)
	if e = writeJSON(filepath.Join(s.Root, id, "session.json"), d.Session); e != nil {
		return e
	}
	return nil
}
func (s *Store) End(id string) error {
	s.pending.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	d, e := s.load(id)
	if e != nil {
		return e
	}
	d.Session.Ended = time.Now().UnixMilli()
	e = writeJSON(filepath.Join(s.Root, id, "session.json"), d.Session)
	delete(s.live, id)
	return e
}
func (s *Store) Delete(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		if !valid(id) {
			return errors.New("Invalid server session identifier")
		}
		if s.live[id] != nil {
			return errors.New("End the connection before deleting its server session")
		}
	}
	for _, id := range ids {
		if e := os.RemoveAll(filepath.Join(s.Root, id)); e != nil {
			return e
		}
		delete(s.cache, id)
	}
	return nil
}

func (s *Store) ResourcePacks(id string) ([]protocol.StackResourcePack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, e := s.load(id)
	if e != nil {
		return nil, e
	}
	out := []protocol.StackResourcePack{}
	for i := len(d.Events) - 1; i >= 0; i-- {
		if d.Events[i].Kind != "packs" {
			continue
		}
		var p packet.ResourcePackStack
		if json.Unmarshal(d.Events[i].Data, &p) == nil {
			out = append(out, p.TexturePacks...)
		}
		break
	}
	return out, nil
}

func (s *Store) Import(source Document) (Session, error) {
	if (source.Format != "craftbox-server-archive" && source.Format != "craftbox-command-archive") || source.Version != 1 || source.Events == nil {
		return Session{}, errors.New("Choose a CraftBox server archive JSON file")
	}
	if len(source.Events) > 30000 {
		return Session{}, errors.New("Server archive exceeds the event limit")
	}
	filtered := []Event{}
	for _, event := range source.Events {
		if dialogEvent(event.Kind) {
			continue
		}
		filtered = append(filtered, event)
	}
	source.Events = filtered
	for i := range source.Events {
		if err := validateEvent(&source.Events[i]); err != nil {
			return Session{}, err
		}
	}
	m, e := s.Begin(source.Session.Name, "", source.Session.Protocol)
	if e != nil {
		return m, e
	}
	s.mu.Lock()
	d := s.live[m.ID]
	f, e := os.OpenFile(filepath.Join(s.Root, m.ID, "events.jsonl"), os.O_CREATE|os.O_WRONLY, 0600)
	if e == nil {
		for i, event := range source.Events {
			event.ID = strconv.Itoa(i + 1)
			event.Packet = ""
			line, err := json.Marshal(event)
			if err != nil {
				e = err
				break
			}
			if d.Session.Bytes+int64(len(line)) > 128<<20 {
				e = errors.New("Server archive exceeds 128 MiB")
				break
			}
			if _, e = f.Write(append(line, '\n')); e != nil {
				break
			}
			apply(d, event)
			d.Session.Bytes += int64(len(line) + 1)
		}
		ce := f.Close()
		if e == nil {
			e = ce
		}
	}
	s.mu.Unlock()
	endErr := s.End(m.ID)
	if e == nil {
		e = endErr
	}
	if e != nil {
		_ = s.Delete([]string{m.ID})
		return m, e
	}
	dCopy, e := s.Get(m.ID)
	return dCopy.Session, e
}
