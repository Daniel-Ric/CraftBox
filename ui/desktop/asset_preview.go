package desktop

import (
	"fmt"

	"github.com/Daniel-Ric/CraftBox/internal/scene"
)

func (a *App) ApplyAssetStack(worldID string, ids []string) error {
	if len(ids) > 50 {
		return fmt.Errorf("Preview up to 50 resource packs")
	}
	packs := []scene.PackPreview{}
	items, e := a.assets().List()
	if e != nil {
		return e
	}
	for _, id := range ids {
		found := false
		for _, v := range items {
			if v.ID == id && v.Kind == "packs" {
				source, e := a.assets().PreviewSource(id)
				if e != nil {
					return e
				}
				packs = append(packs, scene.PackPreview{Name: v.Name, Files: source})
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Selected resource pack is no longer available")
		}
	}
	a.mu.Lock()
	doc := a.renderDocumentLocked(worldID)
	if doc != nil && doc.info.ID == worldID {
		a.mu.Unlock()
		doc.exportMu.Lock()
		defer doc.exportMu.Unlock()
		if doc.project != nil {
			return fmt.Errorf("Open the original Bedrock world to change its resource packs")
		}
		if doc.store == nil {
			return fmt.Errorf("World is no longer open")
		}
		return doc.store.PreviewPacks(a.ctx, packs)
	}
	store := a.scenes[worldID]
	a.mu.Unlock()
	if store == nil {
		return fmt.Errorf("Open the world in 3D before applying resource packs")
	}
	return store.PreviewPacks(a.ctx, packs)
}
func (a *App) TraceSceneResource(worldID string, position [3]int32, face int) (scene.ResourceTrace, error) {
	a.mu.Lock()
	doc := a.renderDocumentLocked(worldID)
	if doc != nil && doc.info.ID == worldID {
		a.mu.Unlock()
		doc.exportMu.Lock()
		defer doc.exportMu.Unlock()
		if doc.store == nil {
			return scene.ResourceTrace{}, fmt.Errorf("Open the original Bedrock world to inspect block resources")
		}
		return doc.store.TraceResource(a.ctx, position, face)
	}
	store := a.scenes[worldID]
	a.mu.Unlock()
	if store == nil {
		return scene.ResourceTrace{}, fmt.Errorf("World is no longer open")
	}
	return store.TraceResource(a.ctx, position, face)
}
