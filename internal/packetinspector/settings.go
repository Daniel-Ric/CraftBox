package packetinspector

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type preferences struct {
	Settings Settings `json:"settings"`
	Rules    []Rule   `json:"rules"`
}

func (c *Controller) loadPreferences() {
	f, e := os.Open(filepath.Join(c.Store.Root, "preferences.json"))
	if e != nil {
		return
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil || info.Size() > 16<<20 {
		return
	}
	var p preferences
	if json.NewDecoder(f).Decode(&p) != nil || p.Settings.MaxDiskMB < 16 || p.Settings.MaxDiskMB > 1048576 {
		return
	}
	if c.Engine.SetRules(p.Rules) != nil {
		return
	}
	p.Settings.Intervention = false
	c.settings = p.Settings
	c.Store.MaxBytes = p.Settings.MaxDiskMB << 20
}
func (c *Controller) savePreferences() error {
	if e := os.MkdirAll(c.Store.Root, 0700); e != nil {
		return e
	}
	settings := c.settings
	settings.Intervention = false
	value := preferences{Settings: settings, Rules: c.Engine.Rules()}
	raw, e := json.Marshal(value)
	if e != nil {
		return e
	}
	if len(raw) > 16<<20 {
		return errors.New("Saved rules exceed 16 MiB")
	}
	return atomicJSON(filepath.Join(c.Store.Root, "preferences.json"), value)
}
func (c *Controller) SetRules(rules []Rule) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	previous := c.Engine.Rules()
	if e := c.Engine.SetRules(rules); e != nil {
		return e
	}
	if e := c.savePreferences(); e != nil {
		_ = c.Engine.SetRules(previous)
		return e
	}
	return nil
}
