package savedsettings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const MaxRecent = 10

type Mode struct {
	Address string `json:"address,omitempty"`

	Options map[string]string `json:"options,omitempty"`
}

type Saved struct {
	Modes map[string]Mode `json:"modes,omitempty"`

	Recent []string `json:"recent_servers,omitempty"`
}

func Load(path string) (*Saved, error) {
	s := &Saved{}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return &Saved{}, err
	}

	recent := s.Recent
	s.Recent = nil
	for i := len(recent) - 1; i >= 0; i-- {
		s.AddRecent(recent[i])
	}
	return s, nil
}

func (s *Saved) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func (s *Saved) Mode(mode string) (Mode, bool) {
	m, ok := s.Modes[mode]
	return m, ok
}

func (s *Saved) SetMode(mode string, m Mode) {
	if s.Modes == nil {
		s.Modes = make(map[string]Mode)
	}
	m.Address = strings.TrimSpace(m.Address)
	s.Modes[mode] = m
}

func (s *Saved) AddRecent(address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		return
	}
	s.Recent = slices.DeleteFunc(s.Recent, func(a string) bool {
		return strings.EqualFold(a, address)
	})
	s.Recent = slices.Insert(s.Recent, 0, address)
	if len(s.Recent) > MaxRecent {
		s.Recent = s.Recent[:MaxRecent]
	}
}

func (s *Saved) RemoveRecent(address string) {
	address = strings.TrimSpace(address)
	s.Recent = slices.DeleteFunc(s.Recent, func(a string) bool {
		return strings.EqualFold(a, address)
	})
}
