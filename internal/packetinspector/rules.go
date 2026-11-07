package packetinspector

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
)

type Rule struct {
	PreMS      int    `json:"preMS"`
	PostMS     int    `json:"postMS"`
	CooldownMS int    `json:"cooldownMS"`
	ID         string `json:"id"`
	Label      string `json:"label"`
	Enabled    bool   `json:"enabled"`
	Name       string `json:"name"`
	Direction  string `json:"direction"`
	Search     string `json:"search"`
	Action     string `json:"action"`
	Payload    string `json:"payload"`
	DelayMS    int    `json:"delayMS"`
	Hits       uint64 `json:"hits"`
}
type Decision struct {
	PreMS      int
	PostMS     int
	CooldownMS int
	Action     string `json:"action"`
	Payload    string `json:"payload"`
	DelayMS    int    `json:"delayMS"`
	Rule       string `json:"rule"`
}
type Engine struct {
	mu      sync.Mutex
	enabled bool
	rules   []Rule
}

func NewEngine() *Engine { return &Engine{} }
func (e *Engine) Interested(name, direction string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range e.rules {
		if r.Enabled && (r.Name == "" || strings.EqualFold(r.Name, name)) && (r.Direction == "" || r.Direction == direction) && e.enabled && r.Action != "bookmark" && r.Action != "trigger" {
			return true
		}
	}
	return false
}
func (e *Engine) Enable(v bool) { e.mu.Lock(); e.enabled = v; e.mu.Unlock() }
func (e *Engine) SetRules(rules []Rule) error {
	if len(rules) > 64 {
		return errors.New("Use at most 64 packet rules")
	}
	for _, r := range rules {
		if r.PreMS < 0 || r.PreMS > 30000 || r.PostMS < 0 || r.PostMS > 30000 || r.CooldownMS < 0 || r.CooldownMS > 3600000 {
			return errors.New("Trigger windows must be 0–30 seconds; cooldown up to one hour")
		}
		if len(r.ID) > 100 || len(r.Search) > 2048 || len(r.Name) > 256 || len(r.Payload) > 1<<20 || r.DelayMS < 0 || r.DelayMS > 5000 {
			return errors.New("Rule exceeds limits")
		}
		if r.Direction != "" && r.Direction != "serverbound" && r.Direction != "clientbound" {
			return errors.New("Invalid rule direction")
		}
		switch r.Action {
		case "block", "edit", "delay", "bookmark", "trigger":
		default:
			return errors.New("Unknown rule action")
		}
		if r.Action == "edit" && !json.Valid([]byte(r.Payload)) {
			return errors.New("Edited packet must be valid JSON")
		}
		if strings.TrimSpace(r.Name) == "" && strings.TrimSpace(r.Search) == "" {
			return errors.New("A rule needs a packet type or search condition")
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append([]Rule{}, rules...)
	return nil
}
func (e *Engine) Rules() []Rule {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]Rule{}, e.rules...)
}
func (e *Engine) Summaries() []Rule {
	rules := e.Rules()
	for i := range rules {
		rules[i].Payload = ""
	}
	return rules
}
func (e *Engine) Decide(p Packet) Decision {
	return e.evaluate(p, false)
}
func (e *Engine) Observe(p Packet) Decision { return e.evaluate(p, true) }
func (e *Engine) evaluate(p Packet, passive bool) Decision {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range e.rules {
		r := &e.rules[i]
		if (r.Action == "bookmark" || r.Action == "trigger") != passive {
			continue
		}
		if !r.Enabled || r.Name != "" && !strings.EqualFold(r.Name, p.Name) || r.Direction != "" && r.Direction != p.Direction || !Matches(p, r.Search) {
			continue
		}
		if r.Action != "bookmark" && r.Action != "trigger" && !e.enabled {
			continue
		}
		r.Hits++
		return Decision{Action: r.Action, Payload: r.Payload, DelayMS: r.DelayMS, Rule: r.ID, PreMS: r.PreMS, PostMS: r.PostMS, CooldownMS: r.CooldownMS}
	}
	return Decision{Action: "forward"}
}
func (e *Engine) HasTriggers() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range e.rules {
		if r.Enabled && r.Action == "trigger" {
			return true
		}
	}
	return false
}
