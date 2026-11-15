package desktop

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WorldFileDetails struct {
	Bytes         int64  `json:"bytes"`
	Modified      string `json:"modified"`
	ResourcePacks int    `json:"resourcePacks"`
	BehaviorPacks int    `json:"behaviorPacks"`
}

func worldImportArchive(world SavedWorld) (string, error) {
	if world.State != "finished" {
		return "", fmt.Errorf("finish saving this world before importing it into Minecraft")
	}
	if !strings.EqualFold(filepath.Ext(world.Path), ".mcworld") {
		return "", fmt.Errorf("this world has no saved .mcworld file")
	}
	path, err := filepath.Abs(world.Path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("saved world is unavailable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("saved world is not a file")
	}
	return path, nil
}

func (a *App) importableWorld(id string) (SavedWorld, string, error) {
	a.mu.Lock()
	var selected *SavedWorld
	for _, world := range a.state.Worlds {
		if world.ID == id {
			copy := world
			selected = &copy
			break
		}
	}
	a.mu.Unlock()
	if selected == nil {
		return SavedWorld{}, "", fmt.Errorf("world not found")
	}
	path, err := worldImportArchive(*selected)
	return *selected, path, err
}

func (a *App) ImportWorldInMinecraft(id string) error {
	_, path, err := a.importableWorld(id)
	if err != nil {
		return err
	}
	return openResultDirectory(path)
}

func (a *App) GetWorldFileDetails(id string) (WorldFileDetails, error) {
	world, path, err := a.importableWorld(id)
	if err != nil {
		return WorldFileDetails{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return WorldFileDetails{}, err
	}
	details := WorldFileDetails{Bytes: info.Size(), Modified: info.ModTime().Format(time.RFC3339)}
	folder := world.CaptureFolder
	if folder == "" {
		folder = strings.TrimSuffix(path, filepath.Ext(path))
	}
	for name, count := range map[string]*int{"world_resource_packs.json": &details.ResourcePacks, "world_behavior_packs.json": &details.BehaviorPacks} {
		file, err := os.Open(filepath.Join(folder, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return details, err
		}
		var packs []json.RawMessage
		err = json.NewDecoder(io.LimitReader(file, 1<<20)).Decode(&packs)
		file.Close()
		if err != nil {
			return details, fmt.Errorf("could not read world packs: %w", err)
		}
		*count = len(packs)
	}
	return details, nil
}
