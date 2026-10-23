//go:build gui && !android

package desktop

import (
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/google/uuid"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type OpenedStudioProject struct {
	World    RenderWorld `json:"world"`
	Document string      `json:"document"`
	Path     string      `json:"path"`
}

func (a *App) SaveStudioProject(worldID, name, document string, saveAs bool) (string, error) {
	if len(document) > maxProjectJSON || !json.Valid([]byte(document)) {
		return "", fmt.Errorf("Invalid project")
	}
	a.mu.Lock()
	path := a.studioProjectPath
	a.mu.Unlock()
	var err error
	if path == "" || saveAs {
		name = strings.Map(func(r rune) rune {
			if r < 32 || strings.ContainsRune("<>:\"/\\|?*", r) {
				return '_'
			}
			return r
		}, name)
		if name == "" {
			name = "Untitled"
		}
		path, err = wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Save complete CraftBox project", DefaultFilename: name + ".craftbox", Filters: []wruntime.FileFilter{{DisplayName: "CraftBox project", Pattern: "*.craftbox"}}})
		if err != nil || path == "" {
			return "", err
		}
		if !strings.EqualFold(filepath.Ext(path), ".craftbox") {
			path += ".craftbox"
		}
	}
	err = writeStudioArchive(path, json.RawMessage(document), func(request scene.PageRequest) (scene.Page, error) { return a.GetRenderScenePage(worldID, request) })
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	a.studioProjectPath = path
	a.mu.Unlock()
	return path, nil
}
func (a *App) OpenStudioProject() (OpenedStudioProject, error) {
	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Open CraftBox project", Filters: []wruntime.FileFilter{{DisplayName: "CraftBox project", Pattern: "*.craftbox"}}})
	if err != nil || path == "" {
		return OpenedStudioProject{}, err
	}
	input, err := os.Open(path)
	if err != nil {
		return OpenedStudioProject{}, err
	}
	defer input.Close()
	stat, err := input.Stat()
	if err != nil || stat.Size() > 8<<30 {
		return OpenedStudioProject{}, fmt.Errorf("Project is too large or unreadable")
	}
	temporary, err := os.MkdirTemp("", "craftbox-project-")
	if err != nil {
		return OpenedStudioProject{}, err
	}
	adopted := false
	defer func() {
		if !adopted {
			os.RemoveAll(temporary)
		}
	}()
	copyPath := filepath.Join(temporary, "document.craftbox")
	output, err := os.Create(copyPath)
	if err != nil {
		return OpenedStudioProject{}, err
	}
	_, err = io.Copy(output, renderContextReader{ctx: a.ctx, reader: input})
	closeErr := output.Close()
	if err != nil {
		return OpenedStudioProject{}, err
	}
	if closeErr != nil {
		return OpenedStudioProject{}, closeErr
	}
	archive, err := readStudioArchive(copyPath)
	if err != nil {
		return OpenedStudioProject{}, err
	}
	info := RenderWorld{ID: "project-" + uuid.NewString(), Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), Chunks: archive.Total}
	a.mu.Lock()
	old := a.pendingStudioDocument
	a.pendingStudioDocument = &renderDocument{info: info, temporary: temporary, project: archive}
	a.pendingStudioPath = path
	a.mu.Unlock()
	adopted = true
	closeRenderDocument(old)
	return OpenedStudioProject{World: info, Document: string(archive.Document), Path: path}, nil
}
func (a *App) SetStudioProjectDirty(dirty bool) {
	a.mu.Lock()
	a.studioProjectDirty = dirty
	a.mu.Unlock()
}
func (a *App) ConfirmStudioProjectClose() {
	a.mu.Lock()
	a.studioCloseApproved = true
	a.mu.Unlock()
	wruntime.Quit(a.ctx)
}
