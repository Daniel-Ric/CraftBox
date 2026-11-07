package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"net/http"
	"time"
)

func (a *App) GetMapPage(id string, r scene.MapRequest) (scene.MapPage, error) {
	return a.getMapPage(a.ctx, id, r)
}
func (a *App) getMapPage(ctx context.Context, id string, r scene.MapRequest) (scene.MapPage, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	a.mu.Lock()
	doc := a.renderDocumentLocked(id)
	if doc != nil && doc.info.ID == id {
		a.mu.Unlock()
		doc.exportMu.Lock()
		defer doc.exportMu.Unlock()
		if doc.store == nil {
			return scene.MapPage{}, fmt.Errorf("This project contains rendered geometry only; use Rendered map")
		}
		return doc.store.MapPage(ctx, r, nil)
	}
	a.mu.Unlock()
	store, parent, scope, keep, err := a.sceneSelection(id)
	if err != nil {
		return scene.MapPage{}, err
	}
	result, err := store.MapPage(ctx, r, keep)
	a.mu.Lock()
	stable := a.scenes[parent] == store && a.sceneResets[parent] == scope
	a.mu.Unlock()
	if !stable {
		return scene.MapPage{}, fmt.Errorf("Map changed; reload the map")
	}
	return result, err
}
func (a *App) serveMapPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", 405)
		return
	}
	var p struct {
		WorldID string           `json:"worldId"`
		Page    scene.MapRequest `json:"page"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		http.Error(w, "Invalid map request", 400)
		return
	}
	page, err := a.getMapPage(r.Context(), p.WorldID, p.Page)
	if err != nil {
		http.Error(w, err.Error(), 409)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(page)
}
