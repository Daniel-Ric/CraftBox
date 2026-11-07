//go:build gui && !android

package desktop

import (
	"github.com/Daniel-Ric/CraftBox/internal/modelexport"
	"github.com/Daniel-Ric/CraftBox/utils"
	"path/filepath"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ExportModel(request ModelExportRequest) (string, error) {
	request.defaults()
	if err := request.Options.Validate(); err != nil {
		return "", err
	}
	path, err := a.defaultModelPath(request.WorldID)
	if err != nil {
		return "", err
	}
	extension, _ := modelexport.Extension(request.Format)
	name := strings.TrimSuffix(filepath.Base(path), ".glb")
	if request.Name != "" {
		name = utils.MakeValidFilename(filepath.Base(request.Name))
	}
	path, err = wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export 3D model", DefaultDirectory: filepath.Dir(path), DefaultFilename: name + extension, Filters: []wruntime.FileFilter{{DisplayName: strings.ToUpper(request.Format) + " model", Pattern: "*" + extension}}})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(path), extension) {
		path += extension
	}
	if err := a.exportModelTo(a.ctx, request, path); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) ExportRenderGLB(id string) (string, error) {
	path, err := a.defaultModelPath(id)
	if err != nil {
		return "", err
	}
	path, err = wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export 3D map as GLB", DefaultDirectory: filepath.Dir(path), DefaultFilename: filepath.Base(path), Filters: []wruntime.FileFilter{{DisplayName: "glTF binary 3D model", Pattern: "*.glb"}}})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".glb") {
		path += ".glb"
	}
	if err := a.exportSceneTo(a.ctx, id, path); err != nil {
		return "", err
	}
	return path, nil
}
