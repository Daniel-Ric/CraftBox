package packetinspector

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/klauspost/compress/s2"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxRecord = 32 << 20
const indexWidth = 32

var validID = regexp.MustCompile(`^[a-f0-9]{32}$`)

type archive struct {
	readCancel  <-chan struct{}
	types       map[string]*TypeStat
	used        time.Time
	meta        Capture
	data, index *os.File
	annotations map[string]Annotation
}
type Store struct {
	Root       string
	MaxBytes   int64
	mu         sync.Mutex
	archives   map[string]*archive
	readers    map[string]int
	readCancel map[string]chan struct{}
	readDone   map[string]chan struct{}
	deleting   map[string]bool
}

func New(root string) *Store {
	return &Store{Root: root, MaxBytes: 4 << 30, archives: map[string]*archive{}, readers: map[string]int{}, readCancel: map[string]chan struct{}{}, readDone: map[string]chan struct{}{}, deleting: map[string]bool{}}
}
func atomicJSON(path string, value any) error {
	raw, e := json.Marshal(value)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), "inspector-*.tmp")
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
	return os.Rename(f.Name(), path)
}
func (s *Store) Create(name, source string, protocol int) (Capture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var id [16]byte
	if _, e := rand.Read(id[:]); e != nil {
		return Capture{}, e
	}
	c := Capture{ID: hex.EncodeToString(id[:]), Name: strings.TrimSpace(name), Source: source, Protocol: protocol, Started: time.Now().UnixMilli()}
	if c.Name == "" {
		c.Name = "NetherNet capture"
	}
	dir := filepath.Join(s.Root, c.ID)
	if e := os.MkdirAll(dir, 0700); e != nil {
		return c, e
	}
	if e := atomicJSON(filepath.Join(dir, "capture.json"), c); e != nil {
		return c, e
	}
	_, e := s.open(c.ID)
	return c, e
}
func (s *Store) open(id string) (*archive, error) {
	if !validID.MatchString(id) {
		return nil, errors.New("Invalid capture ID")
	}
	if a := s.archives[id]; a != nil {
		a.used = time.Now()
		return a, nil
	}
	if len(s.archives) >= 32 {
		oldest := ""
		for key, value := range s.archives {
			if oldest == "" || value.used.Before(s.archives[oldest].used) {
				oldest = key
			}
		}
		a := s.archives[oldest]
		if e := s.save(a); e != nil {
			return nil, e
		}
		a.data.Close()
		a.index.Close()
		delete(s.archives, oldest)
	}
	dir := filepath.Join(s.Root, id)
	info, e := os.Lstat(dir)
	if e != nil {
		return nil, e
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("Capture path is not a regular directory")
	}
	raw, e := os.ReadFile(filepath.Join(dir, "capture.json"))
	if e != nil {
		return nil, e
	}
	var c Capture
	if json.Unmarshal(raw, &c) != nil || c.ID != id {
		return nil, errors.New("Capture metadata is invalid")
	}
	d, e := os.OpenFile(filepath.Join(dir, "packets.bin"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	idx, e := os.OpenFile(filepath.Join(dir, "packets.idx"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		d.Close()
		return nil, e
	}
	di, _ := d.Stat()
	ii, _ := idx.Stat()
	n := ii.Size() / indexWidth
	var row [indexWidth]byte
	for n > 0 {
		if _, e = idx.ReadAt(row[:], (n-1)*indexWidth); e != nil {
			break
		}
		offset := int64(binary.LittleEndian.Uint64(row[:]))
		length := int64(binary.LittleEndian.Uint32(row[8:]))
		if offset >= 0 && length > 0 && length <= maxRecord && offset+length+8 <= di.Size() {
			break
		}
		n--
	}
	end := int64(0)
	if n > 0 {
		end = int64(binary.LittleEndian.Uint64(row[:])) + int64(binary.LittleEndian.Uint32(row[8:])) + 8
	}
	if e = idx.Truncate(n * indexWidth); e != nil {
		d.Close()
		idx.Close()
		return nil, e
	}
	if e = d.Truncate(end); e != nil {
		d.Close()
		idx.Close()
		return nil, e
	}
	if c.Count != uint64(n) {
		c.Bytes = 0
		var ix [indexWidth]byte
		for i := int64(0); i < n; i++ {
			if _, e := idx.ReadAt(ix[:], i*indexWidth); e != nil {
				d.Close()
				idx.Close()
				return nil, e
			}
			c.Bytes += int64(binary.LittleEndian.Uint32(ix[28:]))
		}
	}
	c.Count = uint64(n)
	c.DiskBytes = end + n*indexWidth
	a := &archive{meta: c, data: d, index: idx, annotations: map[string]Annotation{}, used: time.Now()}
	if c.Count == 0 {
		a.types = map[string]*TypeStat{}
	}
	if b, e := os.ReadFile(filepath.Join(dir, "annotations.json")); e == nil && len(b) <= 4<<20 {
		_ = json.Unmarshal(b, &a.annotations)
	}
	s.archives[id] = a
	return a, nil
}
func (s *Store) save(a *archive) error {
	return atomicJSON(filepath.Join(s.Root, a.meta.ID, "capture.json"), a.meta)
}
func (s *Store) Append(id string, p Packet) (Packet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleting[id] {
		return p, errors.New("This recording is being deleted")
	}
	a, e := s.open(id)
	if e != nil {
		return p, e
	}
	p.ID = strconv.FormatUint(a.meta.Count+1, 10)
	if p.Time == 0 {
		p.Time = time.Now().UnixMilli()
	}
	if p.Direction != "serverbound" && p.Direction != "clientbound" {
		return p, errors.New("Invalid packet direction")
	}
	if p.Name == "" || len(p.Name) > 256 {
		return p, errors.New("Invalid packet name")
	}
	if p.Bytes == 0 {
		p.Bytes = len(p.Raw)
	}
	if p.Bytes < 0 {
		return p, errors.New("Packet byte count cannot be negative")
	}
	if a.meta.Count == 0 {
		a.meta.Started = p.Time
	}
	stored := p
	// Decoded is a derived view. Keep one original payload and wire body on disk.
	stored.Decoded = ""
	stored.Summary = ""
	raw, e := json.Marshal(stored)
	if e != nil {
		return p, e
	}
	if len(raw) > maxRecord {
		a.meta.Dropped++
		a.meta.Error = "Packet exceeded the 32 MiB record limit"
		_ = s.save(a)
		return p, errors.New(a.meta.Error)
	}
	compressed := s2.Encode(nil, raw)
	if a.meta.DiskBytes+int64(len(compressed)+8+indexWidth) > s.MaxBytes {
		a.meta.Dropped++
		a.meta.Error = "Capture disk limit reached; recording has gaps"
		_ = s.save(a)
		return p, errors.New(a.meta.Error)
	}
	offset, e := a.data.Seek(0, io.SeekEnd)
	if e != nil {
		return p, e
	}
	var head [8]byte
	binary.LittleEndian.PutUint32(head[:], uint32(len(compressed)))
	binary.LittleEndian.PutUint32(head[4:], crc32.ChecksumIEEE(compressed))
	if _, e = a.data.Write(append(head[:], compressed...)); e != nil {
		return p, e
	}
	var index [indexWidth]byte
	binary.LittleEndian.PutUint64(index[:], uint64(offset))
	binary.LittleEndian.PutUint32(index[8:], uint32(len(compressed)))
	binary.LittleEndian.PutUint32(index[12:], p.PacketID)
	binary.LittleEndian.PutUint64(index[16:], uint64(p.Time))
	binary.LittleEndian.PutUint32(index[28:], uint32(p.Bytes))
	if p.Direction == "serverbound" {
		index[24] = 1
	}
	if p.Action != "" && p.Action != "forward" {
		index[25] = 1
	}
	if _, e = a.index.WriteAt(index[:], int64(a.meta.Count)*indexWidth); e != nil {
		_ = a.data.Truncate(offset)
		return p, e
	}
	a.meta.Count++
	if a.types != nil {
		addTypeStat(a.types, p)
	}
	a.meta.Bytes += int64(p.Bytes)
	a.meta.DiskBytes += int64(len(compressed) + 8 + indexWidth)
	if a.meta.Count%256 == 0 {
		e = s.save(a)
	}
	return p, e
}
func (s *Store) read(a *archive, n uint64) (Packet, error) {
	var p Packet
	select {
	case <-a.readCancel:
		return p, errors.New("Recording read cancelled because the recording is being deleted")
	default:
	}
	if n < 1 || n > a.meta.Count {
		return p, errors.New("Packet is no longer available")
	}
	var row [indexWidth]byte
	if _, e := a.index.ReadAt(row[:], int64(n-1)*indexWidth); e != nil {
		return p, e
	}
	offset := int64(binary.LittleEndian.Uint64(row[:]))
	length := binary.LittleEndian.Uint32(row[8:])
	if length > maxRecord {
		return p, errors.New("Record length is invalid")
	}
	raw := make([]byte, int(length)+8)
	if _, e := a.data.ReadAt(raw, offset); e != nil {
		return p, e
	}
	if binary.LittleEndian.Uint32(raw[:]) != length || crc32.ChecksumIEEE(raw[8:]) != binary.LittleEndian.Uint32(raw[4:]) {
		return p, errors.New("Packet checksum mismatch")
	}
	decodedLength, e := s2.DecodedLen(raw[8:])
	if e != nil || decodedLength > maxRecord {
		return p, errors.New("Packet exceeds decode limit")
	}
	body, e := s2.Decode(nil, raw[8:])
	if e != nil {
		return p, e
	}
	e = json.Unmarshal(body, &p)
	if note, ok := a.annotations[p.ID]; ok {
		p.Note = note.Note
		p.Bookmarked = note.Bookmarked
	}
	return p, e
}
func (s *Store) Detail(id, sequence string, reveal bool) (Packet, error) {
	a, e := s.snapshot(id)
	if e != nil {
		return Packet{}, e
	}
	defer s.closeSnapshot(&a)
	n, e := strconv.ParseUint(sequence, 10, 64)
	if e != nil {
		return Packet{}, e
	}
	p, e := s.read(&a, n)
	if e != nil {
		return Packet{}, e
	}
	// Decode a view outside the writer lock so opening a large packet cannot
	// stall incoming observations or metadata polling.
	p.Decoded = readablePayload(p)
	if !reveal {
		p = Redact(p)
	}
	return p, nil
}

func (s *Store) Capture(id string) (Capture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, e := s.open(id)
	if e != nil {
		return Capture{}, e
	}
	return a.meta, nil
}
func (s *Store) List() ([]Capture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Capture{}
	entries, e := os.ReadDir(s.Root)
	if os.IsNotExist(e) {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	for _, entry := range entries {
		if !entry.IsDir() || !validID.MatchString(entry.Name()) {
			continue
		}
		if a := s.archives[entry.Name()]; a != nil {
			out = append(out, a.meta)
			continue
		}
		raw, e := os.ReadFile(filepath.Join(s.Root, entry.Name(), "capture.json"))
		if e != nil || len(raw) > 1<<20 {
			continue
		}
		var c Capture
		if json.Unmarshal(raw, &c) == nil && c.ID == entry.Name() {
			if index, e := os.Stat(filepath.Join(s.Root, c.ID, "packets.idx")); e == nil {
				c.Count = uint64(index.Size() / indexWidth)
			}
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started > out[j].Started })
	return out, nil
}
func (s *Store) Finish(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, e := s.open(id)
	if e != nil {
		return e
	}
	a.meta.Ended = time.Now().UnixMilli()
	_ = a.data.Sync()
	_ = a.index.Sync()
	return s.save(a)
}
func (s *Store) Gap(id string, count uint64, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, e := s.open(id); e == nil {
		a.meta.Dropped += count
		a.meta.Error = reason
		_ = s.save(a)
	}
}
func (s *Store) Update(u CaptureUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, e := s.open(u.Capture)
	if e != nil {
		return e
	}
	if u.Packet != "" {
		n, e := strconv.ParseUint(u.Packet, 10, 64)
		if e != nil || n < 1 || n > a.meta.Count {
			return errors.New("Invalid annotation packet")
		}
		if len(u.Annotation.Note) > 8192 || len(a.annotations) >= 10000 {
			return errors.New("Annotation limit reached")
		}
		a.annotations[u.Packet] = u.Annotation
		return atomicJSON(filepath.Join(s.Root, u.Capture, "annotations.json"), a.annotations)
	}
	if len(u.Name) > 160 || len(u.Notes) > 8192 {
		return errors.New("Capture name or notes too long")
	}
	if strings.TrimSpace(u.Name) != "" {
		a.meta.Name = strings.TrimSpace(u.Name)
	}
	a.meta.Notes = u.Notes
	return s.save(a)
}
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	if s.deleting[id] {
		s.mu.Unlock()
		return errors.New("This recording is already being deleted")
	}
	_, e := s.open(id)
	if e != nil {
		s.mu.Unlock()
		return e
	}
	// Block new snapshots and cancel existing scans, then wait outside the
	// store mutex until their Windows file handles have actually been closed.
	s.deleting[id] = true
	if cancel := s.readCancel[id]; cancel != nil {
		close(cancel)
	}
	done := s.readDone[id]
	s.mu.Unlock()
	if done != nil {
		<-done
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	defer delete(s.deleting, id)
	a, e := s.open(id)
	if e != nil {
		return e
	}
	a.data.Close()
	a.index.Close()
	delete(s.archives, id)
	dir := filepath.Join(s.Root, id)
	if !validID.MatchString(id) || filepath.Dir(dir) != filepath.Clean(s.Root) {
		return fmt.Errorf("Invalid owned capture path")
	}
	return os.RemoveAll(dir)
}
func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, a := range s.archives {
		_ = s.save(a)
		a.data.Close()
		a.index.Close()
		delete(s.archives, id)
	}
}
