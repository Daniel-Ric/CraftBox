package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"io"
	"net/http"
	"strconv"
)

func (a *App) sceneSelection(worldID string) (*scene.Store, string, uint64, func(int32, int32) bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if messages.WorldDeleted(worldID) {
		return nil, worldID, 0, nil, fmt.Errorf("Capture deleted")
	}
	parentID := worldID
	var selected *messages.WorldRegion
	for parent, regions := range a.worldRegions {
		for _, r := range regions {
			if r.ID == worldID {
				copy := r
				selected = &copy
				parentID = parent
			}
		}
	}
	store := a.scenes[parentID]
	scope := a.sceneResets[parentID]
	if store == nil {
		return nil, parentID, scope, nil, fmt.Errorf("No 3D data has been captured for this world yet")
	}
	regions := append([]messages.WorldRegion{}, a.worldRegions[parentID]...)
	var keep func(int32, int32) bool
	if selected != nil {
		keep = selected.Bounds.Contains
	} else if len(regions) > 0 {
		keep = func(x, z int32) bool {
			for _, r := range regions {
				if r.Move && r.Bounds.Contains(x, z) {
					return false
				}
			}
			return true
		}
	}
	return store, parentID, scope, keep, nil
}

func (a *App) GetScenePage(worldID string, request scene.PageRequest) (scene.Page, error) {
	return a.getScenePage(a.ctx, worldID, request)
}
func (a *App) getScenePage(ctx context.Context, worldID string, request scene.PageRequest) (scene.Page, error) {
	store, parent, scope, keep, err := a.sceneSelection(worldID)
	if err != nil {
		return scene.Page{}, err
	}
	if request.Scope != scope {
		request.After = 0
		request.Target = 0
		request.Cursor = nil
		request.Scope = scope
	}
	cacheScope := ""
	if keep != nil {
		cacheScope = worldID + ":" + strconv.FormatUint(scope, 10)
	}
	result, err := store.PageScoped(ctx, request, keep, cacheScope)
	a.mu.Lock()
	stable := a.scenes[parent] == store && a.sceneResets[parent] == scope
	a.mu.Unlock()
	if !stable {
		return scene.Page{}, fmt.Errorf("Regions changed; reloading the 3D view")
	}
	return result, err
}

func (a *App) serveScenePage(out http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		out.Header().Set("Allow", "POST")
		http.Error(out, "POST required", 405)
		return
	}
	var payload struct {
		WorldID string             `json:"worldId"`
		Page    *scene.PageRequest `json:"page"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(out, req.Body, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.WorldID == "" || payload.Page == nil {
		http.Error(out, "World and page request are required", 400)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		http.Error(out, "Invalid page request", 400)
		return
	}
	result, err := a.getScenePage(req.Context(), payload.WorldID, *payload.Page)
	if err != nil {
		if req.Context().Err() != nil {
			return
		}
		http.Error(out, err.Error(), 409)
		return
	}
	out.Header().Set("Cache-Control", "no-store")
	out.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(out).Encode(result)
}
