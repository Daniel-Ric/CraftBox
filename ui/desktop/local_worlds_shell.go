//go:build gui && !android

package desktop

import (
	"path/filepath"
	"strings"

	"github.com/Daniel-Ric/CraftBox/internal/localworlds"
	"github.com/Daniel-Ric/CraftBox/utils"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) PickLocalWorldFolder() (LocalWorldCatalog, error) {
	folder, err := wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Choose a Bedrock world or Minecraft storage folder"})
	if err != nil {
		return LocalWorldCatalog{}, err
	}
	if folder == "" {
		return a.GetLocalWorlds()
	}
	return a.registerLocalWorldFolder(folder)
}
func (a *App) PickLocalWorldPackFolder(id string) error {
	if _, err := a.localWorld(id); err != nil {
		return err
	}
	folder, err := wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Choose the missing pack or a folder containing packs"})
	if err != nil || folder == "" {
		return err
	}
	return a.registerLocalWorldPackFolder(id, folder)
}
func (a *App) OpenLocalWorldFolder(id string) error {
	w, err := a.localWorld(id)
	if err != nil {
		return err
	}
	return a.OpenResultFolder(w.Folder)
}
func (a *App) ExportLocalWorld(request LocalWorldRequest) (*localworlds.Result, error) {
	return a.runLocalWorldExport(request, func(name string) (string, error) {
		suffix := ""
		if request.Options.BackupOnly {
			suffix = " backup"
		}
		path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export singleplayer world with its packs", DefaultFilename: utils.MakeValidFilename(name+suffix) + ".mcworld", Filters: []wruntime.FileFilter{{DisplayName: "Minecraft Bedrock world", Pattern: "*.mcworld"}}})
		if err != nil || path == "" {
			return path, err
		}
		if !strings.EqualFold(filepath.Ext(path), ".mcworld") {
			path += ".mcworld"
		}
		return path, nil
	})
}
