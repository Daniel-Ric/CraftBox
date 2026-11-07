package desktop

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Daniel-Ric/CraftBox/internal/modelexport"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/utils"
)

type ModelExportStatus struct {
	scene.GLBProgress
	Running bool   `json:"running"`
	JobID   string `json:"jobId,omitempty"`
}
type ModelExportRequest struct {
	JobID   string        `json:"jobId,omitempty"`
	Bounds  *scene.Bounds `json:"bounds,omitempty"`
	WorldID string        `json:"worldId"`
	Name    string        `json:"name,omitempty"`
	modelexport.Options
}

func (r *ModelExportRequest) defaults() {
	if r.Format == "" {
		r.Format = "glb"
		if r.Scale == 0 {
			r.Scale = 1
		}
	}
}

type modelSource struct {
	name, path string
	write      func(context.Context, io.Writer, string, func(scene.GLBProgress)) error
}

func (a *App) modelSource(id string) (modelSource, error) {
	return a.modelSourceBounds(id, nil)
}
func (a *App) modelSourceBounds(id string, bounds *scene.Bounds) (modelSource, error) {
	if bounds != nil {
		if err := bounds.Validate(); err != nil {
			return modelSource{}, err
		}
	}
	a.mu.Lock()
	doc := a.renderDocumentLocked(id)
	if doc != nil && doc.info.ID == id {
		name, path := doc.info.Name, doc.sourcePath
		a.mu.Unlock()
		return modelSource{name: name, path: path, write: func(ctx context.Context, out io.Writer, dir string, progress func(scene.GLBProgress)) error {
			doc.exportMu.Lock()
			defer doc.exportMu.Unlock()
			if doc.project != nil {
				return scene.WriteGLBPages(ctx, out, dir, func(request scene.PageRequest) (scene.Page, error) {
					request.Bounds = bounds
					return doc.project.Page(request)
				}, progress)
			}
			if doc.store == nil {
				return fmt.Errorf("Imported world geometry is unavailable")
			}
			var keep func(int32, int32) bool
			if bounds != nil {
				keep = bounds.Contains
			}
			return doc.store.WriteGLB(ctx, out, dir, keep, true, progress)
		}}, nil
	}
	var name, path string
	for _, w := range a.state.Worlds {
		if w.ID == id {
			name, path = w.Name, w.Path
			break
		}
	}
	a.mu.Unlock()
	store, _, _, keep, err := a.sceneSelection(id)
	if err != nil {
		return modelSource{}, err
	}
	if name == "" {
		return modelSource{}, fmt.Errorf("Unknown world")
	}
	if bounds != nil {
		previous := keep
		keep = func(x, z int32) bool { return bounds.Contains(x, z) && (previous == nil || previous(x, z)) }
	}
	return modelSource{name: name, path: path, write: func(ctx context.Context, out io.Writer, dir string, progress func(scene.GLBProgress)) error {
		return store.WriteGLB(ctx, out, dir, keep, false, progress)
	}}, nil
}

func (a *App) GetModelExportProgress() ModelExportStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.modelExportStatus
}
func (a *App) CancelGLBExport() {
	a.mu.Lock()
	cancel := a.modelExportCancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// CancelModelExportJob never lets one window cancel another window's job.
func (a *App) CancelModelExportJob(jobID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if jobID != "" && a.modelExportStatus.JobID == jobID && a.modelExportCancel != nil {
		a.modelExportCancel()
	}
}

func (a *App) exportSceneTo(ctx context.Context, id, path string) error {
	return a.exportModelTo(ctx, ModelExportRequest{WorldID: id, Options: modelexport.Options{Format: "glb", Scale: 1}}, path)
}
func (a *App) exportModelTo(ctx context.Context, request ModelExportRequest, path string) error {
	request.defaults()
	if err := request.Options.Validate(); err != nil {
		return err
	}
	id := request.WorldID
	extension, _ := modelexport.Extension(request.Format)
	if err := ctx.Err(); err != nil {
		return err
	}
	source, err := a.modelSourceBounds(id, request.Bounds)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(strings.ToLower(path), extension) {
		return fmt.Errorf("Choose a %s destination", extension)
	}
	a.mu.Lock()
	if a.modelExportCancel != nil {
		a.mu.Unlock()
		return fmt.Errorf("Another model export is already running")
	}
	ctx, cancel := context.WithCancel(ctx)
	a.modelExportCancel = cancel
	a.modelExportStatus = ModelExportStatus{Running: true, JobID: request.JobID}
	a.mu.Unlock()
	defer func() {
		cancel()
		a.mu.Lock()
		a.modelExportCancel = nil
		a.modelExportStatus.Running = false
		a.mu.Unlock()
	}()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".craftbox-model-*.glb")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	defer f.Close()
	progress := func(p scene.GLBProgress) {
		a.mu.Lock()
		a.modelExportStatus = ModelExportStatus{GLBProgress: p, Running: true, JobID: request.JobID}
		a.mu.Unlock()
	}
	if err := source.write(ctx, f, dir, progress); err != nil {
		return err
	}
	output, err := os.CreateTemp(dir, ".craftbox-export-*")
	if err != nil {
		return err
	}
	defer os.Remove(output.Name())
	defer output.Close()
	if err := modelexport.Convert(ctx, f, output, request.Options, func(done, total int) {
		progress(scene.GLBProgress{Stage: "converting", Done: done, Total: total})
	}); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := output.Sync(); err != nil {
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return os.Rename(output.Name(), path)
}

func (a *App) exportModelDefault(ctx context.Context, request ModelExportRequest) (string, error) {
	request.defaults()
	if err := request.Options.Validate(); err != nil {
		return "", err
	}
	path, err := a.defaultModelPath(request.WorldID)
	if err != nil {
		return "", err
	}
	name := strings.TrimSuffix(filepath.Base(path), ".glb")
	if request.Name != "" {
		name = utils.MakeValidFilename(filepath.Base(request.Name))
	}
	extension, _ := modelexport.Extension(request.Format)
	path = filepath.Join(filepath.Dir(path), name+extension)
	if err := a.exportModelTo(ctx, request, path); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) defaultModelPath(id string) (string, error) {
	source, err := a.modelSource(id)
	if err != nil {
		return "", err
	}
	if source.path != "" {
		path := strings.TrimRight(source.path, "/\\")
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path + ".glb", nil
		}
		return strings.TrimSuffix(path, filepath.Ext(path)) + ".glb", nil
	}
	return utils.PathData("models", utils.MakeValidFilename(source.name)+"--"+utils.MakeValidFilename(id)+".glb"), nil
}
