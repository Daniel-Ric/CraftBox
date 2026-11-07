//go:build gui && !android

package desktop

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

func saveMapPNG(ctx context.Context, name, data string) (string, error) {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(data, prefix) || len(data) > 192<<20 {
		return "", fmt.Errorf("Map PNG is too large")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, prefix))
	if err != nil {
		return "", err
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 16384 || cfg.Height > 16384 || int64(cfg.Width)*int64(cfg.Height) > 64<<20 {
		return "", fmt.Errorf("Map PNG exceeds the pixel budget")
	}
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "" || name == "." {
		name = "map"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".png") {
		name += ".png"
	}
	path, err := wruntime.SaveFileDialog(ctx, wruntime.SaveDialogOptions{Title: "Export map", DefaultFilename: name, Filters: []wruntime.FileFilter{{DisplayName: "PNG map", Pattern: "*.png"}}})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".png") {
		path += ".png"
	}
	if err = os.WriteFile(path, raw, 0600); err != nil {
		return "", err
	}
	return path, nil
}
func (a *App) SaveMapPNG(name, data string) (string, error) { return saveMapPNG(a.ctx, name, data) }
func (a *MapWindowApp) SaveMapPNG(name, data string) (string, error) {
	return saveMapPNG(a.ctx, name, data)
}
