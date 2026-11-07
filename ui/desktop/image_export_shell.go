//go:build gui && !android

package desktop

import (
	"context"
	"github.com/Daniel-Ric/CraftBox/internal/exportimage"
	"github.com/Daniel-Ric/CraftBox/utils"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"path/filepath"
	"strings"
)

func saveImage(ctx context.Context, request exportimage.Request) (string, error) {
	raw, err := exportimage.Encode(request)
	if err != nil {
		return "", err
	}
	extension, err := exportimage.Extension(request.Format)
	if err != nil {
		return "", err
	}
	name := utils.MakeValidFilename(filepath.Base(strings.ReplaceAll(request.Name, "\\", "/")))
	if name == "" || name == "." {
		name = "CraftBox image"
	}
	name = strings.TrimSuffix(name, filepath.Ext(name)) + extension
	path, err := wruntime.SaveFileDialog(ctx, wruntime.SaveDialogOptions{Title: "Export image", DefaultFilename: name, Filters: []wruntime.FileFilter{{DisplayName: strings.ToUpper(request.Format) + " image", Pattern: "*" + extension}}})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), extension) {
		path += extension
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".craftbox-image-*")
	if err != nil {
		return "", err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	defer file.Close()
	if _, err = file.Write(raw); err != nil {
		return "", err
	}
	if err = file.Sync(); err != nil {
		return "", err
	}
	if err = file.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(temporary, path); err != nil {
		return "", err
	}
	return path, nil
}
func (a *App) SaveImage(request exportimage.Request) (string, error) {
	return saveImage(a.ctx, request)
}
func (a *MapWindowApp) SaveImage(request exportimage.Request) (string, error) {
	return saveImage(a.ctx, request)
}
