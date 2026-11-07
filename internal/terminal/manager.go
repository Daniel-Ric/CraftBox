package terminal

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

type Request struct {
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
	Cwd  string `json:"cwd"`
}
type Info struct {
	ID    string `json:"id"`
	PID   int    `json:"pid"`
	Shell string `json:"shell"`
	Cwd   string `json:"cwd"`
}
type Output struct {
	Data     string `json:"data"`
	Exited   bool   `json:"exited"`
	ExitCode int    `json:"exitCode"`
}
type Process interface {
	io.ReadWriteCloser
	Resize(int, int) error
	Wait(context.Context) (int, error)
}
type Factory func(Request) (Process, Info, error)
type session struct {
	readMu    sync.Mutex
	remainder []byte
	process   Process
	chunks    chan []byte
	done      chan struct{}
	cancel    context.CancelFunc
	input     sync.Mutex
	mu        sync.Mutex
	exited    bool
	code      int
	once      sync.Once
}
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
	factory  Factory
	closed   bool
}

func New() *Manager { return NewWithFactory(startPowerShell) }
func NewWithFactory(factory Factory) *Manager {
	return &Manager{sessions: map[string]*session{}, factory: factory}
}
func validSize(c, r int) bool { return c >= 2 && c <= 500 && r >= 2 && r <= 300 }
func (m *Manager) Create(r Request) (Info, error) {
	if !validSize(r.Cols, r.Rows) {
		return Info{}, errors.New("terminal size must be 2–500 columns and 2–300 rows")
	}
	if r.Cwd == "" {
		r.Cwd, _ = os.Getwd()
	}
	abs, err := filepath.Abs(r.Cwd)
	if err != nil {
		return Info{}, err
	}
	r.Cwd = abs
	stat, err := os.Stat(r.Cwd)
	if err != nil || !stat.IsDir() {
		return Info{}, fmt.Errorf("terminal working directory is unavailable")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return Info{}, errors.New("terminal manager is closed")
	}
	if len(m.sessions) >= 8 {
		return Info{}, errors.New("at most eight PowerShell terminals can be open")
	}
	p, info, err := m.factory(r)
	if err != nil {
		return Info{}, err
	}
	info.ID = uuid.NewString()
	info.Cwd = r.Cwd
	ctx, cancel := context.WithCancel(context.Background())
	s := &session{process: p, chunks: make(chan []byte, 64), done: make(chan struct{}), cancel: cancel}
	m.sessions[info.ID] = s
	go func() {
		defer close(s.done)
		buf := make([]byte, 4096)
		for {
			n, err := p.Read(buf)
			if n > 0 {
				chunk := append([]byte(nil), buf[:n]...)
				select {
				case s.chunks <- chunk:
				case <-ctx.Done():
				}
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		code, _ := p.Wait(ctx)
		s.mu.Lock()
		s.exited = true
		s.code = code
		s.mu.Unlock()
		s.once.Do(func() { _ = p.Close(); <-s.done; cancel() })
	}()
	return info, nil
}
func (m *Manager) get(id string) (*session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[id]
	if s == nil {
		return nil, errors.New("terminal is closed or unknown")
	}
	return s, nil
}
func (m *Manager) Read(id string) (Output, error) {
	s, err := m.get(id)
	if err != nil {
		return Output{}, err
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	var data []byte
	if len(s.remainder) > 0 {
		data = append(data, s.remainder...)
		s.remainder = nil
	}
	for len(data) < 64<<10 {
		select {
		case chunk := <-s.chunks:
			n := min(len(chunk), (64<<10)-len(data))
			data = append(data, chunk[:n]...)
			if n < len(chunk) {
				s.remainder = chunk[n:]
			}
		default:
			goto ready
		}
	}
ready:
	s.mu.Lock()
	out := Output{Data: base64.StdEncoding.EncodeToString(data), ExitCode: s.code}
	exited := s.exited
	s.mu.Unlock()
	select {
	case <-s.done:
		out.Exited = exited && len(s.chunks) == 0 && len(s.remainder) == 0
	default:
	}
	return out, nil
}
func (m *Manager) Write(id, data string) error {
	if len(data) > 64<<10 {
		return errors.New("terminal input exceeds 64 KiB")
	}
	s, err := m.get(id)
	if err != nil {
		return err
	}
	s.input.Lock()
	defer s.input.Unlock()
	_, err = io.WriteString(s.process, data)
	return err
}
func (m *Manager) Resize(id string, c, r int) error {
	if !validSize(c, r) {
		return errors.New("invalid terminal dimensions")
	}
	s, err := m.get(id)
	if err != nil {
		return err
	}
	return s.process.Resize(c, r)
}
func (m *Manager) Close(id string) error {
	m.mu.Lock()
	s := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if s == nil {
		return nil
	}
	s.cancel()
	s.once.Do(func() { _ = s.process.Close() })
	return nil
}
func (m *Manager) Shutdown() {
	m.mu.Lock()
	m.closed = true
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.Close(id)
	}
}
