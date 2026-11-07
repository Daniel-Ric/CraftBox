package diagnostics

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const MiB int64 = 1024 * 1024
const segmentBytes int64 = 4 * MiB
const segmentCount = 3
const CrashBytes int64 = 2 * MiB
const sessionReserve = segmentBytes*segmentCount + CrashBytes + 64*1024

var sessionName = regexp.MustCompile(`^session-[0-9]{8}T[0-9]{6}Z-[0-9]+-[0-9a-f]{12}$`)
var fileName = regexp.MustCompile(`^(events-[0-9]{6}\.jsonl|crash\.log|session\.json)$`)
var current atomic.Pointer[Manager]
var configuredStore atomic.Pointer[Store]

func SetStore(s *Store) { configuredStore.Store(s) }
func AvailableStore() *Store {
	if m := Current(); m != nil {
		return m.Store()
	}
	return configuredStore.Load()
}
func (s *Store) ReportError(err error) { s.warningError(err) }

type Policy struct {
	MaxBytes    int64  `json:"maxBytes"`
	MaxAgeDays  int    `json:"maxAgeDays"`
	MaxSessions int    `json:"maxSessions"`
	Level       string `json:"level"`
}

func DefaultPolicy() Policy { return Policy{100 * MiB, 14, 20, "debug"} }
func (p Policy) validate() error {
	if p.MaxBytes < 32*MiB || p.MaxBytes > 1024*MiB {
		return errors.New("log budget must be between 32 and 1024 MiB")
	}
	if p.MaxAgeDays < 1 || p.MaxAgeDays > 90 || p.MaxSessions < 1 || p.MaxSessions > 100 {
		return errors.New("retention must be 1-90 days and 1-100 sessions")
	}
	if p.Level != "info" && p.Level != "debug" && p.Level != "trace" {
		return errors.New("invalid log detail level")
	}
	return nil
}

type Options struct {
	Version, Role string
	SegmentBytes  int64
	Segments      int
}
type File struct {
	Name     string `json:"name"`
	Bytes    int64  `json:"bytes"`
	Modified string `json:"modified"`
}
type Session struct {
	ID      string `json:"id"`
	Started string `json:"started"`
	Ended   string `json:"ended,omitempty"`
	Version string `json:"version"`
	Role    string `json:"role"`
	PID     int    `json:"pid"`
	Clean   bool   `json:"clean"`
	Fatal   bool   `json:"fatal,omitempty"`
	Status  string `json:"status"`
	Active  bool   `json:"active"`
	Bytes   int64  `json:"bytes"`
	Files   []File `json:"files"`
}
type Status struct {
	Directory     string    `json:"directory"`
	Policy        Policy    `json:"policy"`
	Bytes         int64     `json:"bytes"`
	ReservedBytes int64     `json:"reservedBytes"`
	Sessions      []Session `json:"sessions"`
	Warning       string    `json:"warning,omitempty"`
}
type Preview struct {
	Text      string `json:"text"`
	Truncated bool   `json:"truncated"`
}
type Store struct {
	root    *os.Root
	mu      sync.Mutex
	policy  Policy
	warning string
}

func OpenStore(path string) (*Store, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	s := &Store{root: root, policy: DefaultPolicy()}
	s.refreshPolicy()
	return s, nil
}
func (s *Store) Close() error   { return s.root.Close() }
func (s *Store) Path() string   { return s.root.Name() }
func (s *Store) Policy() Policy { s.mu.Lock(); defer s.mu.Unlock(); return s.policy }
func (s *Store) refreshPolicy() {
	raw, err := readSmall(s.root, "policy.json", 8192)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var p Policy
	if err != nil || len(raw) > 8192 || json.Unmarshal(raw, &p) != nil || p.validate() != nil {
		s.warning = "Log settings could not be read; defaults are in use."
		return
	}
	s.policy = p
}
func (s *Store) warningError(err error) {
	if err != nil {
		s.mu.Lock()
		s.warning = Redact(err.Error())
		s.mu.Unlock()
	}
}
func (s *Store) storeLock() (*os.File, error) {
	f, e := s.root.OpenFile(".retention.lock", os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	for i := 0; i < 100; i++ {
		if lockFile(f) == nil {
			return f, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	f.Close()
	return nil, errors.New("log storage is busy; try again")
}
func activeIn(root *os.Root) bool {
	for _, name := range []string{".active.lock", ".crash.lock"} {
		f, e := root.OpenFile(name, os.O_RDWR, 0600)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return true
		}
		e = lockFile(f)
		f.Close()
		if e != nil {
			return true
		}
	}
	return false
}
func (s *Store) sessions() ([]Session, error) {
	entries, e := fs.ReadDir(s.root.FS(), ".")
	if e != nil {
		return nil, e
	}
	result := []Session{}
	for _, entry := range entries {
		if !entry.IsDir() || !sessionName.MatchString(entry.Name()) {
			continue
		}
		root, e := s.root.OpenRoot(entry.Name())
		if e != nil {
			continue
		}
		raw, e := readSmall(root, "session.json", 16384)
		var session Session
		if e != nil || len(raw) > 16384 || json.Unmarshal(raw, &session) != nil || session.ID != entry.Name() {
			info, _ := entry.Info()
			started := time.Now().UTC().Format(time.RFC3339Nano)
			if info != nil {
				started = info.ModTime().UTC().Format(time.RFC3339Nano)
			}
			session = Session{ID: entry.Name(), Started: started, Version: "unknown (incomplete session metadata)"}
		}
		session.Active = activeIn(root)
		session.Files = []File{}
		files, _ := fs.ReadDir(root.FS(), ".")
		for _, f := range files {
			if !fileName.MatchString(f.Name()) {
				continue
			}
			info, e := root.Lstat(f.Name())
			if e != nil || !info.Mode().IsRegular() {
				continue
			}
			session.Files = append(session.Files, File{f.Name(), info.Size(), info.ModTime().UTC().Format(time.RFC3339Nano)})
			session.Bytes += info.Size()
		}
		session.Status = "interrupted"
		if session.Clean {
			session.Status = "finished"
		}
		if !session.Clean && session.Fatal {
			session.Status = "crashed"
		}
		for _, f := range session.Files {
			if f.Name == "crash.log" && f.Bytes > 0 {
				session.Status = "crashed"
			}
		}
		if session.Active {
			session.Status = "active"
		}
		result = append(result, session)
		root.Close()
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Started > result[j].Started })
	return result, nil
}
func (s *Store) Status() (Status, error) {
	sessions, e := s.sessions()
	p := s.Policy()
	status := Status{Directory: s.Path(), Policy: p, Sessions: sessions}
	for _, session := range sessions {
		status.Bytes += session.Bytes
		if session.Active {
			status.ReservedBytes += sessionReserve
		}
	}
	s.mu.Lock()
	status.Warning = s.warning
	s.mu.Unlock()
	return status, e
}
func (s *Store) SetPolicy(p Policy) error {
	if err := p.validate(); err != nil {
		return err
	}
	lock, err := s.storeLock()
	if err != nil {
		return err
	}
	defer lock.Close()
	status, err := s.Status()
	if err != nil {
		return err
	}
	if p.MaxBytes < status.ReservedBytes {
		return fmt.Errorf("active sessions reserve %.0f MiB; stop other CraftBox instances before reducing the budget", float64(status.ReservedBytes)/float64(MiB))
	}
	if err := s.prunePolicy(p); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(p, "", "  ")
	if err = s.root.WriteFile("policy.tmp", raw, 0600); err != nil {
		return err
	}
	if err = s.root.Rename("policy.tmp", "policy.json"); err != nil {
		return err
	}
	s.mu.Lock()
	s.policy = p
	s.mu.Unlock()
	return nil
}
func (s *Store) pruneLocked() error {
	return s.prunePolicy(s.Policy())
}
func (s *Store) prunePolicy(p Policy) error {
	sessions, err := s.sessions()
	if err != nil {
		return err
	}
	var bytes int64
	remaining := len(sessions)
	for _, v := range sessions {
		if v.Active {
			bytes += sessionReserve
		} else {
			bytes += v.Bytes
		}
	}
	for i := len(sessions) - 1; i >= 0; i-- {
		v := sessions[i]
		if v.Active {
			continue
		}
		started, _ := time.Parse(time.RFC3339Nano, v.Started)
		if bytes <= p.MaxBytes && remaining <= p.MaxSessions && time.Since(started) <= time.Duration(p.MaxAgeDays)*24*time.Hour {
			continue
		}
		if err := s.root.RemoveAll(v.ID); err != nil {
			return err
		}
		bytes -= v.Bytes
		remaining--
	}
	return nil
}
func (s *Store) Cleanup() error {
	lock, e := s.storeLock()
	if e != nil {
		return e
	}
	defer lock.Close()
	s.refreshPolicy()
	return s.pruneLocked()
}
func (s *Store) Delete(id string) error {
	if !sessionName.MatchString(id) {
		return errors.New("invalid log session")
	}
	lock, e := s.storeLock()
	if e != nil {
		return e
	}
	defer lock.Close()
	root, e := s.root.OpenRoot(id)
	if e != nil {
		return e
	}
	active := activeIn(root)
	root.Close()
	if active {
		return errors.New("active sessions cannot be deleted")
	}
	return s.root.RemoveAll(id)
}
func (s *Store) DeleteClosed() error {
	lock, e := s.storeLock()
	if e != nil {
		return e
	}
	defer lock.Close()
	sessions, e := s.sessions()
	if e != nil {
		return e
	}
	for _, v := range sessions {
		if !v.Active {
			if e = s.root.RemoveAll(v.ID); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *Store) Read(id, name string) (Preview, error) {
	if !sessionName.MatchString(id) || !fileName.MatchString(name) {
		return Preview{}, errors.New("invalid log file")
	}
	f, e := s.root.Open(filepath.Join(id, name))
	if e != nil {
		return Preview{}, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() {
		return Preview{}, errors.New("not a regular log file")
	}
	const maxPreview int64 = 128 * 1024
	offset := max(int64(0), info.Size()-maxPreview)
	data, e := io.ReadAll(io.NewSectionReader(f, offset, maxPreview))
	if e != nil {
		return Preview{}, e
	}
	if offset > 0 {
		if i := strings.IndexByte(string(data), '\n'); i >= 0 {
			data = data[i+1:]
		}
	}
	return Preview{Redact(string(data)), offset > 0}, nil
}
func (s *Store) FilePath(id, name string) (string, error) {
	if _, e := s.Read(id, name); e != nil {
		return "", e
	}
	return filepath.Join(s.Path(), id, name), nil
}

// SessionLogSnapshot combines the retained event segments of exactly one session.
// Opening only the latest segment could hide the error after a log rotation.
func (s *Store) SessionLogSnapshot(id string) (result string, err error) {
	if !sessionName.MatchString(id) {
		return "", errors.New("invalid log session")
	}
	root, err := s.root.OpenRoot(id)
	if err != nil {
		return "", err
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return "", err
	}
	var names []string
	for _, entry := range entries {
		candidate := entry.Name()
		if !entry.IsDir() && fileName.MatchString(candidate) && strings.HasPrefix(candidate, "events-") && strings.HasSuffix(candidate, ".jsonl") {
			names = append(names, candidate)
		}
	}
	if len(names) == 0 {
		return "", errors.New("This session has no saved event log")
	}
	sort.Strings(names)
	output, err := os.CreateTemp("", id+"-*.log")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = output.Close()
		if err != nil {
			_ = os.Remove(output.Name())
		}
	}()
	if _, err = fmt.Fprintf(output, "CraftBox session: %s\n\n", id); err != nil {
		return "", err
	}
	for _, name := range names {
		file, openErr := root.Open(name)
		if openErr != nil {
			return "", openErr
		}
		info, statErr := file.Stat()
		if statErr != nil || !info.Mode().IsRegular() {
			file.Close()
			return "", errors.New("not a regular log file")
		}
		_, err = fmt.Fprintf(output, "--- %s ---\n", name)
		if err == nil {
			_, err = io.Copy(output, io.LimitReader(file, segmentBytes))
		}
		file.Close()
		if err != nil {
			return "", err
		}
	}
	if err = output.Close(); err != nil {
		return "", err
	}
	return output.Name(), nil
}

type Manager struct {
	removeSegment func(string) error
	mu            sync.Mutex
	store         *Store
	root          *os.Root
	lock          *os.File
	session       Session
	file          *os.File
	size          int64
	index         int
	segments      int
	segmentBytes  int64
	sequence      uint64
	started       time.Time
	closed        bool
	stop          chan struct{}
	done          chan struct{}
}

func (s *Store) Start(o Options) (*Manager, error) {
	lock, e := s.storeLock()
	if e != nil {
		return nil, e
	}
	defer lock.Close()
	if e = s.pruneLocked(); e != nil {
		return nil, e
	}
	status, e := s.Status()
	if e != nil {
		return nil, e
	}
	if status.ReservedBytes+sessionReserve > s.Policy().MaxBytes {
		return nil, errors.New("log budget is reserved by active sessions; close another CraftBox instance")
	}
	var suffix [6]byte
	if _, e = rand.Read(suffix[:]); e != nil {
		return nil, e
	}
	now := time.Now().UTC()
	id := fmt.Sprintf("session-%s-%d-%s", now.Format("20060102T150405Z"), os.Getpid(), hex.EncodeToString(suffix[:]))
	if e = s.root.Mkdir(id, 0700); e != nil {
		return nil, e
	}
	root, e := s.root.OpenRoot(id)
	if e != nil {
		return nil, e
	}
	active, e := root.OpenFile(".active.lock", os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		root.Close()
		return nil, e
	}
	if e = lockFile(active); e != nil {
		active.Close()
		root.Close()
		return nil, e
	}
	m := &Manager{store: s, root: root, lock: active, session: Session{ID: id, Started: now.Format(time.RFC3339Nano), PID: os.Getpid(), Version: o.Version, Role: o.Role}, segments: o.Segments, segmentBytes: o.SegmentBytes, started: now, stop: make(chan struct{}), done: make(chan struct{})}
	if m.segmentBytes <= 0 {
		m.segmentBytes = segmentBytes
	}
	if m.segments <= 0 {
		m.segments = segmentCount
	}
	m.segmentBytes = min(segmentBytes, max(int64(512), m.segmentBytes))
	m.segments = min(segmentCount, m.segments)
	if e = m.saveMeta(); e != nil {
		active.Close()
		root.Close()
		return nil, e
	}
	if e = s.pruneLocked(); e != nil {
		active.Close()
		root.Close()
		_ = s.root.RemoveAll(id)
		return nil, e
	}
	m.removeSegment = root.Remove
	m.Record("info", "application", "session.started", "Application session started", map[string]any{"version": o.Version, "role": o.Role, "os": runtime.GOOS, "arch": runtime.GOARCH, "go": runtime.Version(), "cpus": runtime.NumCPU()})
	go m.monitor()
	return m, nil
}
func (m *Manager) ID() string    { return m.session.ID }
func (m *Manager) Store() *Store { return m.store }
func SetCurrent(m *Manager)      { current.Store(m) }
func Current() *Manager          { return current.Load() }
func Record(level, component, event, message string, fields map[string]any) {
	if m := Current(); m != nil {
		m.Record(level, component, event, message, fields)
	}
}
func (m *Manager) saveMeta() error {
	raw, _ := json.MarshalIndent(m.session, "", "  ")
	if e := m.root.WriteFile("session.tmp", raw, 0600); e != nil {
		return e
	}
	return m.root.Rename("session.tmp", "session.json")
}
func (m *Manager) Record(level, component, event, message string, fields map[string]any) {
	p := m.store.Policy()
	if level == "trace" && p.Level != "trace" || level == "debug" && p.Level == "info" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	m.sequence++
	if level == "fatal" || level == "panic" {
		m.session.Fatal = true
		m.store.warningError(m.saveMeta())
	}
	entry := map[string]any{"time": time.Now().UTC().Format(time.RFC3339Nano), "session": m.ID(), "sequence": m.sequence, "elapsedMs": time.Since(m.started).Milliseconds(), "level": level, "component": component, "event": event, "message": limited(Redact(message), 16384)}
	if len(fields) > 0 {
		entry["fields"] = safeFields(fields)
	}
	raw, e := json.Marshal(entry)
	if e != nil {
		entry["fields"] = "[unsupported diagnostic fields]"
		raw, _ = json.Marshal(entry)
	}
	if int64(len(raw)+1) > m.segmentBytes || len(raw) > 32768 {
		delete(entry, "fields")
		entry["message"] = "[oversized diagnostic entry omitted] " + limited(Redact(message), 256)
		raw, _ = json.Marshal(entry)
	}
	raw = append(raw, '\n')
	if m.file == nil || m.size+int64(len(raw)) > m.segmentBytes {
		next := m.index + 1
		if expired := next - m.segments; expired > 0 {
			if err := m.removeSegment(fmt.Sprintf("events-%06d.jsonl", expired)); err != nil && !errors.Is(err, os.ErrNotExist) {
				m.store.warningError(fmt.Errorf("Log rotation paused until an old file can be removed: %w", err))
				return
			}
		}
		if m.file != nil {
			m.file.Sync()
			m.file.Close()
			m.file = nil
		}
		m.size = 0
		m.file, e = m.root.OpenFile(fmt.Sprintf("events-%06d.jsonl", next), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			m.store.warningError(e)
			m.file = nil
			return
		}
		m.index = next
	}
	n, e := m.file.Write(raw)
	m.size += int64(n)
	m.store.warningError(e)
	if level == "error" || level == "fatal" || level == "panic" {
		m.file.Sync()
	}
}
func (m *Manager) monitor() {
	defer close(m.done)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	m.sample()
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.sample()
			m.store.warningError(m.store.Cleanup())
		}
	}
}
func (m *Manager) sample() {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	m.Record("info", "runtime", "runtime.sample", "Runtime health", map[string]any{"heapBytes": memory.HeapAlloc, "heapInUseBytes": memory.HeapInuse, "systemBytes": memory.Sys, "goroutines": runtime.NumGoroutine(), "gcCycles": memory.NumGC, "gcPauseTotalNs": memory.PauseTotalNs})
}
func (m *Manager) Snapshot(reason string) {
	buffer := make([]byte, 64*1024)
	n := runtime.Stack(buffer, true)
	stack := Redact(string(buffer[:n]))
	id := time.Now().UTC().Format(time.RFC3339Nano)
	for offset, part := 0, 1; offset < len(stack); offset, part = offset+8000, part+1 {
		m.Record("error", "runtime", "diagnostic.snapshot", reason, map[string]any{"snapshot": id, "part": part, "stack": stack[offset:min(offset+8000, len(stack))], "truncated": n == len(buffer)})
	}
	m.sample()
}
func (m *Manager) Close() {
	m.Record("info", "application", "session.finished", "Application session ended normally", nil)
	m.finish(true)
}
func (m *Manager) abandon() { m.finish(false) }
func (m *Manager) finish(clean bool) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	close(m.stop)
	if m.file != nil {
		m.file.Sync()
		m.file.Close()
	}
	m.session.Clean = clean
	if clean {
		m.session.Ended = time.Now().UTC().Format(time.RFC3339Nano)
	}
	m.store.warningError(m.saveMeta())
	m.lock.Close()
	m.root.Close()
	m.mu.Unlock()
	<-m.done
	_ = m.store.Cleanup()
}

func RecordCrash(input io.Reader, path, id string, limit int64) error {
	return RecordCrashReady(input, path, id, limit, nil)
}
func RecordCrashReady(input io.Reader, path, id string, limit int64, ready func()) error {
	if !sessionName.MatchString(id) {
		return errors.New("invalid crash session")
	}
	root, e := os.OpenRoot(path)
	if e != nil {
		return e
	}
	defer root.Close()
	session, e := root.OpenRoot(id)
	if e != nil {
		return e
	}
	defer session.Close()
	lock, e := session.OpenFile(".crash.lock", os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	defer lock.Close()
	if e = lockFile(lock); e != nil {
		return e
	}
	if ready != nil {
		ready()
	}
	first := make([]byte, 1)
	if _, e = io.ReadFull(input, first); e == io.EOF {
		return nil
	}
	if e != nil {
		return e
	}
	f, e := session.OpenFile("crash.log", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		io.Copy(io.Discard, input)
		return e
	}
	defer f.Close()
	marker := []byte("\n[Crash report truncated at storage limit]\n")
	header := []byte("CraftBox fatal report | " + id + " | " + time.Now().UTC().Format(time.RFC3339Nano) + "\n")
	if limit <= int64(len(header)+len(marker)+1) {
		return errors.New("crash limit too small")
	}
	capacity := limit - int64(len(header)+len(marker))
	data, err := io.ReadAll(io.LimitReader(io.MultiReader(strings.NewReader(string(first)), input), capacity+1))
	if err != nil {
		io.Copy(io.Discard, input)
		return err
	}
	truncated := int64(len(data)) > capacity
	if truncated {
		data = data[:capacity]
	}
	data = []byte(Redact(string(data)))
	if int64(len(data)) > capacity {
		data = data[:capacity]
		truncated = true
	}
	if _, e = f.Write(header); e != nil {
		io.Copy(io.Discard, input)
		return e
	}
	if _, e = f.Write(data); e != nil {
		io.Copy(io.Discard, input)
		return e
	}
	if truncated {
		_, e = f.Write(marker)
	}
	_, drainError := io.Copy(io.Discard, input)
	if e == nil {
		e = drainError
	}
	f.Sync()
	return e
}

func limited(s string, n int) string {
	if len(s) > n {
		return strings.ToValidUTF8(s[:n], "?") + " [truncated]"
	}
	return s
}

var secretKey = regexp.MustCompile(`(?i)(token|password|authorization|secret|content.?key|cookie|auth.?code|skin.?data|payload|api.?key|credential|^key$)`)
var urlValue = regexp.MustCompile(`(?i)[a-z][a-z0-9+.-]*://[^\s"'<>]+`)
var querySecret = regexp.MustCompile(`(?i)(token|password|auth|secret|api.?key|^key$|sig|credential|code)`)
var bearerValue = regexp.MustCompile(`(?i)bearer\s+[^\s"'<>]+`)
var credentialValue = regexp.MustCompile(`(?i)((?:access[_-]?token|refresh[_-]?token|id[_-]?token|token|authorization|password|secret|content[_-]?key|api[_-]?key|auth|user[_-]?code|device[_-]?code|key|code|sig)["']?\s*[:=]\s*["']?)[^&\s,"'<>}]+`)
var jwtValue = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)

func Redact(s string) string {
	s = urlValue.ReplaceAllStringFunc(s, func(raw string) string {
		u, e := url.Parse(raw)
		if e != nil {
			return "[unparseable URL omitted]"
		}
		if u.User != nil {
			u.User = url.User("[redacted]")
		}
		q := u.Query()
		for k := range q {
			if querySecret.MatchString(k) {
				q.Set(k, "[redacted]")
			}
		}
		u.RawQuery = q.Encode()
		return u.String()
	})
	s = bearerValue.ReplaceAllString(s, "Bearer [redacted]")
	s = credentialValue.ReplaceAllString(s, "${1}[redacted]")
	return jwtValue.ReplaceAllString(s, "[redacted]")
}
func safeFields(fields map[string]any) map[string]any { return scrubMap(fields, 0) }
func scrubMap(fields map[string]any, depth int) map[string]any {
	out := map[string]any{}
	if depth > 5 {
		return map[string]any{"truncated": true}
	}
	for k, v := range fields {
		if len(out) >= 40 {
			out["truncatedFields"] = true
			break
		}
		if secretKey.MatchString(k) {
			out[k] = "[redacted]"
		} else {
			out[k] = scrubValue(v, depth+1)
		}
	}
	return out
}
func scrubValue(value any, depth int) any {
	if depth > 5 {
		return "[nested value omitted]"
	}
	switch v := value.(type) {
	case nil, bool, int, int32, int64, uint, uint32, uint64, float32, float64:
		return v
	case string:
		return limited(Redact(v), 8192)
	case error:
		return limited(Redact(v.Error()), 8192)
	case map[string]any:
		return scrubMap(v, depth)
	case []any:
		out := []any{}
		for i, v := range v {
			if i >= 40 {
				out = append(out, "[remaining values omitted]")
				break
			}
			out = append(out, scrubValue(v, depth+1))
		}
		return out
	default:
		raw, e := json.Marshal(value)
		if e != nil {
			return "[unsupported value]"
		}
		if len(raw) > 8192 {
			return "[oversized value omitted]"
		}
		var decoded any
		if json.Unmarshal(raw, &decoded) != nil {
			return "[unsupported value]"
		}
		return scrubValue(decoded, depth+1)
	}
}

func readSmall(root *os.Root, name string, limit int64) ([]byte, error) {
	f, e := root.Open(name)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() {
		return nil, errors.New("invalid diagnostic metadata file")
	}
	data, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(data)) > limit {
		return nil, errors.New("diagnostic metadata exceeds limit")
	}
	return data, nil
}
