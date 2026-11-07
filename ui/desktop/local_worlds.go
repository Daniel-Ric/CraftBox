package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Daniel-Ric/CraftBox/internal/assetlibrary"
	"github.com/Daniel-Ric/CraftBox/internal/localworlds"
	"github.com/google/uuid"
)

type LocalWorldCatalog struct {
	Worlds  []localworlds.World `json:"worlds"`
	Warning string              `json:"warning,omitempty"`
}
type LocalWorldRequest struct {
	WorldID     string              `json:"worldId"`
	JobID       string              `json:"jobId"`
	Fingerprint string              `json:"fingerprint"`
	Options     localworlds.Options `json:"options"`
}

func (a *App) localWorldContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
func (a *App) GetLocalWorlds() (LocalWorldCatalog, error) {
	installations, lookupErr := assetlibrary.InstalledMinecraft()
	roots := []localworlds.Root{}
	for _, installation := range installations {
		if installation.StorageAvailable {
			roots = append(roots, localworlds.Root{Storage: installation.Storage, Name: installation.Name})
		}
	}
	a.mu.Lock()
	roots = append(roots, a.localWorldRoots...)
	a.mu.Unlock()
	worlds, err := localworlds.Discover(a.localWorldContext(), roots)
	if err != nil {
		return LocalWorldCatalog{}, err
	}
	a.mu.Lock()
	if a.localWorldSources == nil {
		a.localWorldSources = map[string]localworlds.World{}
	}
	for i, w := range worlds {
		if previous, ok := a.localWorldSources[w.ID]; ok {
			w.ExtraPackRoots = previous.ExtraPackRoots
			worlds[i] = w
		}
		a.localWorldSources[w.ID] = w
	}
	a.mu.Unlock()
	result := LocalWorldCatalog{Worlds: worlds}
	if lookupErr != nil {
		result.Warning = "Some Minecraft installations could not be identified. You can choose a world folder manually."
	}
	return result, nil
}
func (a *App) localWorld(id string) (localworlds.World, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	w, ok := a.localWorldSources[id]
	if !ok {
		return w, fmt.Errorf("select a local world from the list first")
	}
	return w, nil
}
func (a *App) InspectLocalWorld(request LocalWorldRequest) (localworlds.Plan, error) {
	w, err := a.localWorld(request.WorldID)
	if err != nil {
		return localworlds.Plan{}, err
	}
	return localworlds.Inspect(a.localWorldContext(), w, request.Options)
}
func (a *App) registerLocalWorldFolder(folder string) (LocalWorldCatalog, error) {
	absolute, err := filepath.Abs(folder)
	if err != nil {
		return LocalWorldCatalog{}, err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return LocalWorldCatalog{}, err
	}
	root := localworlds.Root{Name: "Selected folder", Folder: absolute}
	if _, err = os.Stat(filepath.Join(absolute, "minecraftWorlds")); err == nil {
		root.Storage = absolute
		root.Folder = ""
	} else if strings.EqualFold(filepath.Base(absolute), "minecraftWorlds") {
		root.Storage = filepath.Dir(absolute)
	} else if _, err = os.Stat(filepath.Join(absolute, "level.dat")); err == nil {
		root.Storage = filepath.Dir(absolute)
		if strings.EqualFold(filepath.Base(root.Storage), "minecraftWorlds") {
			root.Storage = filepath.Dir(root.Storage)
		}
	} else {
		return LocalWorldCatalog{}, fmt.Errorf("choose a Bedrock world folder, minecraftWorlds, or com.mojang")
	}
	worlds, err := localworlds.Discover(a.localWorldContext(), []localworlds.Root{root})
	if err != nil {
		return LocalWorldCatalog{}, err
	}
	if len(worlds) == 0 {
		return LocalWorldCatalog{}, fmt.Errorf("no Bedrock worlds were found in the selected folder")
	}
	a.mu.Lock()
	found := false
	for _, old := range a.localWorldRoots {
		if old.Storage == root.Storage && old.Folder == root.Folder {
			found = true
			break
		}
	}
	if !found {
		a.localWorldRoots = append(a.localWorldRoots, root)
	}
	a.mu.Unlock()
	return a.GetLocalWorlds()
}
func (a *App) registerLocalWorldPackFolder(id, folder string) error {
	root, err := filepath.Abs(folder)
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("choose a pack folder")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	w, ok := a.localWorldSources[id]
	if !ok {
		return fmt.Errorf("select a local world first")
	}
	for _, old := range w.ExtraPackRoots {
		if strings.EqualFold(root, old) {
			return nil
		}
	}
	w.ExtraPackRoots = append(w.ExtraPackRoots, root)
	a.localWorldSources[id] = w
	return nil
}
func (a *App) GetLocalWorldExportProgress() localworlds.Progress {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.localWorldProgress
}
func (a *App) CancelLocalWorldExportJob(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if id != "" && a.localWorldProgress.JobID == id && a.localWorldCancel != nil {
		a.localWorldCancel()
	}
}
func (a *App) runLocalWorldExport(request LocalWorldRequest, choose func(string) (string, error)) (*localworlds.Result, error) {
	w, err := a.localWorld(request.WorldID)
	if err != nil {
		return nil, err
	}
	if request.JobID == "" {
		request.JobID = uuid.NewString()
	}
	a.mu.Lock()
	if a.localWorldCancel != nil {
		a.mu.Unlock()
		return nil, fmt.Errorf("another singleplayer world export is running")
	}
	ctx, cancel := context.WithCancel(a.localWorldContext())
	a.localWorldCancel = cancel
	a.localWorldProgress = localworlds.Progress{Stage: "Choose export destination", Running: true, JobID: request.JobID}
	a.mu.Unlock()
	defer func() {
		cancel()
		a.mu.Lock()
		a.localWorldCancel = nil
		a.localWorldProgress.Running = false
		a.mu.Unlock()
	}()
	destination, err := choose(w.Name)
	if err != nil {
		return nil, err
	}
	if destination == "" {
		return nil, nil
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	result, err := localworlds.Export(ctx, w, request.Options, destination, request.Fingerprint, func(progress localworlds.Progress) {
		progress.JobID = request.JobID
		a.mu.Lock()
		a.localWorldProgress = progress
		a.mu.Unlock()
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
