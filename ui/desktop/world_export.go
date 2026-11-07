package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Daniel-Ric/CraftBox/internal/worldexport"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/google/uuid"
)

type WorldCopyRequest struct {
	JobID   string              `json:"jobId,omitempty"`
	WorldID string              `json:"worldId"`
	Options worldexport.Options `json:"options"`
}
type WorldCopyResult struct {
	Path   string `json:"path"`
	Folder string `json:"folder"`
	Chunks int    `json:"chunks"`
}
type WorldExportStatus struct {
	worldexport.Progress
	Running bool   `json:"running"`
	JobID   string `json:"jobId,omitempty"`
}

func (a *App) finishedExportWorld(id string) (SavedWorld, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, w := range a.state.Worlds {
		if w.ID != id {
			continue
		}
		if w.State != "finished" {
			return SavedWorld{}, "", fmt.Errorf("wait for this capture to finish before exporting a copy")
		}
		folder := w.CaptureFolder
		if folder == "" {
			folder = strings.TrimSuffix(w.Path, filepath.Ext(w.Path))
		}
		if folder == "" {
			return SavedWorld{}, "", fmt.Errorf("world folder is unavailable")
		}
		return w, folder, nil
	}
	return SavedWorld{}, "", fmt.Errorf("world not found")
}
func (a *App) PreviewWorldExport(request WorldCopyRequest) (worldexport.Plan, error) {
	a.editMu.Lock()
	defer a.editMu.Unlock()
	_, folder, err := a.finishedExportWorld(request.WorldID)
	if err != nil {
		return worldexport.Plan{}, err
	}
	return worldexport.Preview(context.Background(), folder, request.Options)
}

func (a *App) GetWorldExportCatalog(worldID string) (worldexport.Plan, error) {
	a.editMu.Lock()
	defer a.editMu.Unlock()
	_, folder, err := a.finishedExportWorld(worldID)
	if err != nil {
		return worldexport.Plan{}, err
	}
	return worldexport.Describe(context.Background(), folder)
}
func (a *App) GetWorldExportProgress() WorldExportStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.worldExportStatus
}
func (a *App) CancelWorldExport() {
	a.mu.Lock()
	cancel := a.worldExportCancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
func (a *App) CancelWorldExportJob(jobID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if jobID != "" && a.worldExportStatus.JobID == jobID && a.worldExportCancel != nil {
		a.worldExportCancel()
	}
}
func (a *App) ExportWorldCopy(request WorldCopyRequest) (result WorldCopyResult, err error) {
	a.mu.Lock()
	if a.worldExportCancel != nil {
		a.mu.Unlock()
		return result, fmt.Errorf("another world export is running")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	a.worldExportCancel = cancel
	a.worldExportSource = request.WorldID
	a.worldExportStatus = WorldExportStatus{Running: true, JobID: request.JobID}
	a.mu.Unlock()
	defer func() {
		cancel()
		a.mu.Lock()
		a.worldExportCancel = nil
		a.worldExportSource = ""
		a.worldExportStatus.Running = false
		a.mu.Unlock()
	}()
	a.editMu.Lock()
	defer a.editMu.Unlock()
	item, source, err := a.finishedExportWorld(request.WorldID)
	if err != nil {
		return result, err
	}
	plan, err := worldexport.Preview(ctx, source, request.Options)
	if err != nil {
		return result, err
	}
	name := item.Name + " export"
	if request.Options.Settings.Name != nil {
		name = strings.TrimSpace(*request.Options.Settings.Name)
	}
	request.Options.Settings.Name = &name
	parent, err := filepath.Abs(filepath.Dir(source))
	if err != nil {
		return result, err
	}
	destination := filepath.Join(parent, utils.MakeValidFilename(name)+"--"+uuid.NewString())
	if filepath.Dir(destination) != parent {
		return result, fmt.Errorf("invalid export destination")
	}
	progress := func(p worldexport.Progress) {
		a.mu.Lock()
		a.worldExportStatus = WorldExportStatus{Progress: p, Running: true, JobID: request.JobID}
		a.mu.Unlock()
	}
	if err = worldexport.Copy(ctx, source, destination, request.Options, progress); err != nil {
		return result, err
	}
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(destination)
			os.Remove(destination + ".mcworld")
		}
	}()
	progress(worldexport.Progress{Stage: "archive", Total: 1})
	if err = compositionArchive(ctx, destination, destination+".mcworld"); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	progress(worldexport.Progress{Stage: "complete", Completed: 1, Total: 1})
	committed = true
	return WorldCopyResult{Path: destination + ".mcworld", Folder: destination, Chunks: plan.KeptChunks}, nil
}
