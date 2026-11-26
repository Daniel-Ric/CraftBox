package offlinerender

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

type Request struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Format string `json:"format"`
	Model  string `json:"model"`
	Config string `json:"config"`
}

func FindBlender() (string, error) {
	if path, err := exec.LookPath("blender"); err == nil {
		return path, nil
	}
	var patterns []string
	switch runtime.GOOS {
	case "windows":
		for _, name := range []string{"ProgramW6432", "ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
			if root := os.Getenv(name); root != "" {
				patterns = append(patterns, filepath.Join(root, "Blender Foundation", "Blender *", "blender.exe"))
			}
		}
	case "darwin":
		patterns = append(patterns, "/Applications/Blender*.app/Contents/MacOS/Blender")
		if home, err := os.UserHomeDir(); err == nil {
			patterns = append(patterns, filepath.Join(home, "Applications", "Blender*.app", "Contents", "MacOS", "Blender"))
		}
	case "linux":
		patterns = append(patterns, "/snap/bin/blender", "/opt/blender*/blender")
	}
	for _, pattern := range patterns {
		paths, _ := filepath.Glob(pattern)
		slices.Reverse(paths)
		for _, path := range paths {
			if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
				return path, nil
			}
		}
	}
	return "", errors.New("Blender was not found. Install Blender 4.3 or newer or add it to PATH.")
}

func (r Request) Validate() error {
	if r.ID == "" || len(r.ID) > 100 || (r.Format != "png" && r.Format != "exr") ||
		len(r.Model) < 16 || len(r.Model) > 384<<20 || !strings.HasPrefix(r.Model, "Z2xURg") ||
		len(r.Config) > 64<<20 {
		return errors.New("Invalid offline render request")
	}
	var dimensions struct{ Width, Height, Samples int }
	if json.Unmarshal([]byte(r.Config), &dimensions) != nil || dimensions.Width < 2 || dimensions.Height < 2 ||
		dimensions.Width > 4096 || dimensions.Height > 4096 || dimensions.Samples < 1 || dimensions.Samples > 4096 {
		return errors.New("Invalid offline render dimensions or samples")
	}
	return nil
}

func Render(ctx context.Context, blender, output string, r Request) error {
	if err := r.Validate(); err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "craftbox-cycles-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	model := filepath.Join(directory, "scene.glb")
	file, err := os.Create(model)
	if err != nil {
		return err
	}
	_, decodeErr := io.Copy(file, base64.NewDecoder(base64.StdEncoding, strings.NewReader(r.Model)))
	if err = errors.Join(decodeErr, file.Close()); err != nil {
		return err
	}
	config := filepath.Join(directory, "scene.json")
	script := filepath.Join(directory, "render.py")
	if err = os.WriteFile(config, []byte(r.Config), 0600); err != nil {
		return err
	}
	if err = os.WriteFile(script, []byte(blenderScript), 0600); err != nil {
		return err
	}
	stage, err := os.CreateTemp(filepath.Dir(output), ".craftbox-cycles-*."+r.Format)
	if err != nil {
		return err
	}
	stagingPath := stage.Name()
	defer os.Remove(stagingPath)
	if err = stage.Close(); err != nil {
		return err
	}
	command := exec.CommandContext(ctx, blender, "--background", "--factory-startup", "--disable-autoexec", "--python-exit-code", "1", "--python", script, "--", model, config, stagingPath, r.Format)
	command.Dir = directory
	log := &renderLog{}
	command.Stdout = log
	command.Stderr = log
	if err = command.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(log.text))
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if info, err := os.Stat(stagingPath); err != nil || info.Size() < 32 {
		return errors.New("Cycles did not produce an image")
	}
	return os.Rename(stagingPath, output)
}

type renderLog struct{ text string }

func (l *renderLog) Write(p []byte) (int, error) {
	const limit = 6000
	if len(p) >= limit {
		l.text = string(p[len(p)-limit:])
	} else {
		l.text += string(p)
		if len(l.text) > limit {
			l.text = l.text[len(l.text)-limit:]
		}
	}
	return len(p), nil
}
