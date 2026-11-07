package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"io"
	"net/http"
	"os"
	"strconv"
)

func (a *App) closeScenes() {
	a.CancelComposition()
	a.mu.Lock()
	stage := a.compositionStage
	a.compositionStage = nil
	a.mu.Unlock()
	if stage != nil {
		os.RemoveAll(stage.folder)
	}
	a.mu.Lock()
	stores := a.scenes
	a.scenes = nil
	a.mu.Unlock()
	for _, store := range stores {
		store.Close()
	}
}

func (a *App) serveScene(out http.ResponseWriter, req *http.Request) {
	out.Header().Set("Cache-Control", "no-store")
	if req.URL.Path == "/scene" {
		if req.Method != http.MethodGet {
			out.Header().Set("Allow", "GET")
			http.Error(out, "GET required", 405)
			return
		}
		after := uint64(0)
		var err error
		if value := req.URL.Query().Get("after"); value != "" {
			after, err = strconv.ParseUint(value, 10, 64)
		}
		id := req.URL.Query().Get("worldId")
		if err != nil || id == "" {
			http.Error(out, "worldId and a valid sequence are required", 400)
			return
		}
		snapshot, err := a.getScene(req.Context(), id, after)
		if err != nil {
			http.Error(out, err.Error(), 409)
			return
		}
		out.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(out).Encode(snapshot)
		return
	}
	if req.Method == http.MethodGet {
		out.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(out).Encode(a.GetModelExportProgress())
		return
	}
	if req.Method == http.MethodDelete {
		if query := req.URL.Query(); query.Has("jobId") {
			a.CancelModelExportJob(query.Get("jobId"))
		} else {
			a.CancelGLBExport()
		}
		out.WriteHeader(204)
		return
	}
	if req.Method != http.MethodPost {
		out.Header().Set("Allow", "GET, POST, DELETE")
		http.Error(out, "POST required", 405)
		return
	}
	var payload ModelExportRequest
	dec := json.NewDecoder(http.MaxBytesReader(out, req.Body, 4096))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil || payload.WorldID == "" {
		http.Error(out, "worldId is required", 400)
		return
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		http.Error(out, "Invalid request", 400)
		return
	}
	payload.defaults()
	if err := payload.Options.Validate(); err != nil {
		http.Error(out, err.Error(), 400)
		return
	}
	path, err := a.exportModelDefault(req.Context(), payload)
	if err != nil {
		http.Error(out, err.Error(), 409)
		return
	}
	out.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(out).Encode(map[string]string{"path": path})
}

func (a *App) GetScene(worldID string, after uint64) (scene.Snapshot, error) {
	return a.getScene(a.ctx, worldID, after)
}

func (a *App) getScene(ctx context.Context, worldID string, after uint64) (scene.Snapshot, error) {
	store, parentID, resetAfter, keep, err := a.sceneSelection(worldID)
	if err != nil {
		return scene.Snapshot{}, err
	}
	if after < resetAfter {
		after = 0
	}
	result, err := store.SnapshotContext(ctx, after, keep)
	a.mu.Lock()
	stable := a.scenes[parentID] == store && a.sceneResets[parentID] == resetAfter
	a.mu.Unlock()
	if !stable {
		return scene.Snapshot{}, fmt.Errorf("Regions changed while building the 3D scene; reloading view")
	}
	return result, err
}

func (a *App) ExportScene(worldID string) (string, error) {
	filename, err := a.defaultModelPath(worldID)
	if err != nil {
		return "", err
	}
	if err = a.exportSceneTo(a.ctx, worldID, filename); err != nil {
		return "", err
	}
	return filename, nil
}
