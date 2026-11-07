//go:build gui && !android

package desktop

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/internal/structure"
	"github.com/google/uuid"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type OpenedStudioProject struct {
	Structure   *structure.Summary `json:"structure,omitempty"`
	WorkspaceID string             `json:"workspaceId,omitempty"`
	World       RenderWorld        `json:"world"`
	Document    string             `json:"document"`
	Path        string             `json:"path"`
}

func (a *App) SaveStudioProject(worldID, name, document string, saveAs bool) (string, error) {
	return a.SaveStudioProjectForTab(worldID, worldID, name, document, saveAs)
}
func (a *App) SaveStudioProjectForTab(tabID, worldID, name, document string, saveAs bool) (string, error) {
	if len(document) > maxProjectJSON || !json.Valid([]byte(document)) {
		return "", fmt.Errorf("Invalid project")
	}
	var selection struct {
		Selection *scene.Bounds `json:"selection"`
	}
	if err := json.Unmarshal([]byte(document), &selection); err != nil {
		return "", err
	}
	if selection.Selection != nil {
		if err := selection.Selection.Validate(); err != nil {
			return "", err
		}
	}
	a.mu.Lock()
	path := a.studioTabPaths[tabID]
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
	err = writeStudioArchive(path, json.RawMessage(document), func(request scene.PageRequest) (scene.Page, error) {
		request.Bounds = selection.Selection
		return a.GetRenderScenePage(worldID, request)
	})
	if err != nil {
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
func (a *App) OpenStudioProject() (OpenedStudioProject, error) {
	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Open CraftBox project", Filters: []wruntime.FileFilter{{DisplayName: "CraftBox project", Pattern: "*.craftbox;*.craftbox-workspace"}}})
	if err != nil || path == "" {
		return OpenedStudioProject{}, err
	}
	if project, err := zip.OpenReader(path); err == nil {
		isStructure := false
		for _, entry := range project.File {
			if entry.Name == "project.json" && entry.UncompressedSize64 <= 1<<20 {
				if in, e := entry.Open(); e == nil {
					var manifest struct {
						Format string `json:"format"`
					}
					_ = json.NewDecoder(io.LimitReader(in, 1<<20)).Decode(&manifest)
					in.Close()
					isStructure = manifest.Format == "craftbox-structure"
				}
				break
			}
		}
		project.Close()
		if isStructure {
			summary, e := a.OpenStructurePath(path)
			if e != nil {
				return OpenedStudioProject{}, e
			}
			return OpenedStudioProject{Structure: &summary, Path: path}, nil
		}
	}
	if strings.EqualFold(filepath.Ext(path), ".craftbox-workspace") {
		info, e := os.Stat(path)
		if e != nil || info.Size() > maxWorkspaceProject {
			return OpenedStudioProject{}, fmt.Errorf("Workspace project is too large or unreadable")
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			return OpenedStudioProject{}, e
		}
		if e = validateWorkspaceProject(string(raw)); e != nil {
			return OpenedStudioProject{}, e
		}
		id := uuid.NewString()
		a.mu.Lock()
		if a.studioTabPaths == nil {
			a.studioTabPaths = map[string]string{}
		}
		a.studioTabPaths[id] = path
		a.mu.Unlock()
		return OpenedStudioProject{WorkspaceID: id, Document: string(raw), Path: path}, nil
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
	a.keepRenderDocumentLocked(&renderDocument{info: info, temporary: temporary, project: archive, sourcePath: path})
	if a.studioTabPaths == nil {
		a.studioTabPaths = map[string]string{}
	}
	a.studioTabPaths[info.ID] = path
	a.mu.Unlock()
	adopted = true
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

func (a *App) ConfirmSessionClose() {
	a.mu.Lock()
	a.sessionCloseApproved = true
	a.studioCloseApproved = true
	a.mu.Unlock()
	wruntime.Quit(a.ctx)
}
