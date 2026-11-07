package desktop

import (
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"strings"
)

// Empty transfer states have no archive. Release their retained rendering data
// without invoking the user-facing deletion workflow or touching the filesystem.
func (a *App) abandonEmptyWorld(id string) {
	if id == "" {
		return
	}
	a.mu.Lock()
	for _, w := range a.state.Worlds {
		if w.ID == id && (w.Path != "" || w.Chunks > 0 || w.State == "finished") {
			a.mu.Unlock()
			return
		}
	}
	ids := map[string]bool{id: true}
	for changed := true; changed; {
		changed = false
		for _, w := range a.state.Worlds {
			if ids[w.ParentID] && !ids[w.ID] {
				ids[w.ID] = true
				changed = true
			}
		}
	}
	kept := a.state.Worlds[:0]
	for _, w := range a.state.Worlds {
		if !ids[w.ID] {
			kept = append(kept, w)
		}
	}
	a.state.Worlds = kept
	var stores []*scene.Store
	for removed := range ids {
		messages.MarkCaptureDiscarded(removed)
		delete(a.worldTiles, removed)
		delete(a.worldRegions, removed)
		delete(a.sceneResets, removed)
		if store := a.scenes[removed]; store != nil {
			stores = append(stores, store)
			delete(a.scenes, removed)
		}
		for key := range a.tileVersions {
			if strings.HasPrefix(key, removed+":") {
				delete(a.tileVersions, key)
			}
		}
	}
	if ids[a.state.ActiveWorldID] {
		a.state.ActiveWorldID = ""
		a.state.HasPlayer = false
	}
	a.tileSequence++
	a.tileResetSequence = a.tileSequence
	emit := a.emit
	a.mu.Unlock()
	for _, store := range stores {
		store.Close()
	}
	if emit != nil {
		emit("desktop:reset-map", id)
	}
	a.publish()
}
