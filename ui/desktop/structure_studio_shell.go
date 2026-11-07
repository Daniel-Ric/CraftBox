//go:build gui && !android

package desktop

import (
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/structure"
	"github.com/Daniel-Ric/CraftBox/utils"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"io"
	"path/filepath"
	"strings"
)

func (a *App) ChooseStructureSource(kind string) (string, error) {
	switch kind {
	case "pack":
		return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Attach original Bedrock pack", Filters: []wruntime.FileFilter{{DisplayName: "Bedrock pack", Pattern: "*.mcpack"}}})
	case "pack-folder":
		return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Attach Bedrock resource or behavior pack folder"})
	case "folder":
		return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Open local Bedrock world folder"})
	case "world":
		return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Open Bedrock world", Filters: []wruntime.FileFilter{{DisplayName: "Bedrock world", Pattern: "*.mcworld"}}})
	case "structure":
		return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Open Structure Studio project or Bedrock structure", Filters: []wruntime.FileFilter{{DisplayName: "Bedrock structures and Structure Studio", Pattern: "*.mcstructure;*.craftbox"}}})
	default:
		return "", fmt.Errorf("Unknown source type")
	}
}
func (a *App) SaveStructureDialog(id string, project, saveAs bool, o structure.ExportOptions) (structure.Summary, error) {
	s, e := a.structureSession(id)
	if e != nil {
		return structure.Summary{}, e
	}
	s.Mu.Lock()
	path, name := s.Path, s.History.Doc.Name
	s.Mu.Unlock()
	extension := ".mcstructure"
	if project {
		extension = ".craftbox"
	}
	if !project || path == "" || saveAs {
		path, e = wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Save " + extension, DefaultFilename: utils.MakeValidFilename(name) + extension, Filters: []wruntime.FileFilter{{DisplayName: extension, Pattern: "*" + extension}}})
		if e != nil {
			return structure.Summary{}, e
		}
		if path == "" {
			return a.GetStructure(id)
		}
		if !strings.EqualFold(filepath.Ext(path), extension) {
			path += extension
		}
	}
	return a.WriteStructureFile(id, path, project, o)
}
func (a *App) ExportStructureWorldDialog(r StructurePlacement) (string, error) {
	if r.Destination == "" {
		s, e := a.structureSession(r.ID)
		if e != nil {
			return "", e
		}
		s.Mu.Lock()
		name := s.History.Doc.Name
		s.Mu.Unlock()
		path, e := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export blueprint to a new Bedrock world copy", DefaultFilename: utils.MakeValidFilename(name) + ".mcworld", Filters: []wruntime.FileFilter{{DisplayName: "Bedrock world", Pattern: "*.mcworld"}}})
		if e != nil || path == "" {
			return "", e
		}
		if !strings.EqualFold(filepath.Ext(path), ".mcworld") {
			path += ".mcworld"
		}
		r.Destination = path
	}
	return a.ExportStructureWorld(r)
}
func (a *App) ExportStructureMaterials(id string, boxes []structure.Bounds) (string, error) {
	m, e := a.GetStructureMaterials(id, boxes)
	if e != nil {
		return "", e
	}
	path, e := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export blueprint materials", DefaultFilename: "materials.csv", Filters: []wruntime.FileFilter{{DisplayName: "CSV spreadsheet", Pattern: "*.csv"}}})
	if e != nil || path == "" {
		return "", e
	}
	if !strings.EqualFold(filepath.Ext(path), ".csv") {
		path += ".csv"
	}
	e = structure.WriteAtomic(path, func(w io.Writer) error { return structure.WriteMaterialsCSV(w, m) })
	return path, e
}
