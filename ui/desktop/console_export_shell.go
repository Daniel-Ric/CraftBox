//go:build gui

package desktop

import (
	"errors"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) SaveConsoleLog(name, text string) (string, error) {
	if len(text) > 8<<20 {
		return "", errors.New("log export exceeds 8 MiB")
	}
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "." || name == "" {
		name = "CraftBox-console"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".log") {
		name += ".log"
	}
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export console log", DefaultFilename: name, Filters: []wruntime.FileFilter{{DisplayName: "Log file", Pattern: "*.log"}}})
	if err != nil || path == "" {
		return "", err
	}
	if err = os.WriteFile(path, []byte(text), 0600); err != nil {
		return "", err
	}
	return path, nil
}
