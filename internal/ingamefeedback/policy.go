package ingamefeedback

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type Policy struct {
	root  string
	once  sync.Once
	mu    sync.Mutex
	state atomic.Pointer[Settings]
}

type Settings struct {
	Enabled           bool `json:"enabled"`
	ConfirmDisconnect bool `json:"confirmDisconnect"`
	ModelPickFeedback bool `json:"modelPickFeedback"`
	ModelPickFocus    bool `json:"modelPickFocus"`
	MenuColors        bool `json:"menuColors"`
}

func Defaults() Settings {
	return Settings{Enabled: true, ConfirmDisconnect: true, ModelPickFeedback: true, ModelPickFocus: true, MenuColors: true}
}

func New(root string) *Policy { return &Policy{root: root} }
func (p *Policy) load() {
	saved := Defaults()
	if raw, err := os.ReadFile(filepath.Join(p.root, "ingame-feedback.json")); err == nil {
		if json.Unmarshal(raw, &saved) != nil {
			saved = Defaults()
		}
	}
	p.state.Store(&saved)
}
func (p *Policy) Settings() Settings { p.once.Do(p.load); return *p.state.Load() }
func (p *Policy) Enabled() bool      { return p.Settings().Enabled }
func (p *Policy) Set(enabled bool) error {
	return p.update(func(settings *Settings) { settings.Enabled = enabled })
}
func (p *Policy) SetSettings(settings Settings) error {
	return p.update(func(current *Settings) { *current = settings })
}
func (p *Policy) update(change func(*Settings)) error {
	p.once.Do(p.load)
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := os.MkdirAll(p.root, 0700); err != nil {
		return err
	}
	settings := *p.state.Load()
	change(&settings)
	raw, _ := json.Marshal(settings)
	f, err := os.CreateTemp(p.root, "ingame-feedback-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	_, err = f.Write(raw)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(name, filepath.Join(p.root, "ingame-feedback.json")); err != nil {
		return err
	}
	p.state.Store(&settings)
	return nil
}
