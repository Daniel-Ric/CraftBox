//go:build gui && !android

package desktop

import (
	"encoding/json"
	"fmt"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"path/filepath"
	"strings"
)

const maxWorkspaceProject = 32 << 20

func validateWorkspaceProject(raw string) error {
	if len(raw) > maxWorkspaceProject {
		return fmt.Errorf("Workspace project exceeds 32 MiB")
	}
	var d struct {
		Format  string `json:"format"`
		Version int    `json:"version"`
		Kind    string `json:"kind"`
		Source  struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"source"`
		View map[string]json.RawMessage `json:"view"`
	}
	if e := json.Unmarshal([]byte(raw), &d); e != nil {
		return e
	}
	if d.Format != "craftbox-workspace" || d.Version != 1 || (d.Kind != "packs" && d.Kind != "skins" && d.Kind != "packets") || d.Source.ID == "" || d.View == nil {
		return fmt.Errorf("Invalid workspace project")
	}
	return nil
}
func (a *App) SaveWorkspaceProject(tabID, name, document string, saveAs bool) (string, error) {
	if err := validateWorkspaceProject(document); err != nil {
		return "", err
	}
	a.mu.Lock()
	path := a.studioTabPaths[tabID]
	a.mu.Unlock()
	if path == "" || saveAs {
		var err error
		name = strings.Map(func(r rune) rune {
			if r < 32 || strings.ContainsRune("<>:\"/\\|?*", r) {
				return '_'
			}
			return r
		}, name)
		if name == "" {
			name = "Untitled"
		}
		path, err = wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Save CraftBox workspace project", DefaultFilename: name + ".craftbox-workspace", Filters: []wruntime.FileFilter{{DisplayName: "CraftBox workspace project", Pattern: "*.craftbox-workspace"}}})
		if err != nil || path == "" {
			return "", err
		}
		if !strings.EqualFold(filepath.Ext(path), ".craftbox-workspace") {
			path += ".craftbox-workspace"
		}
	}
	if err := writeWorkspaceProject(path, document); err != nil {
		return "", err
	}
	a.mu.Lock()
	if a.studioTabPaths == nil {
		a.studioTabPaths = map[string]string{}
	}
	a.studioTabPaths[tabID] = path
	a.mu.Unlock()
	return path, nil
}

func writeWorkspaceProject(path, document string) error {
	if err := validateWorkspaceProject(document); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".craftbox-workspace-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.WriteString(document)
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	return nil
}
