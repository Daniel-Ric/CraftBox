package desktop

import (
	"github.com/Daniel-Ric/CraftBox/internal/assetlibrary"
	"time"
)

func (a *App) modelAssetsChanged(items []assetlibrary.Item, err error) ([]assetlibrary.Item, error) {
	if err == nil {
		a.invalidateModelPicker()
	}
	return items, err
}

func (a *App) GetPreparedAssetObjects(id string) ([]assetlibrary.Object, error) {
	return a.modelObjects(id)
}
func (a *App) GetPreparedAssetDetail(id string) (assetlibrary.Detail, error) {
	sources, err := a.modelSources()
	if err != nil {
		return assetlibrary.Detail{}, err
	}
	for _, source := range sources {
		if source.Item.ID == id {
			item := source.Item
			if item.Tags == nil {
				item.Tags = []string{}
			}
			return assetlibrary.Detail{Partial: true, Item: item, Files: []assetlibrary.File{}, Issues: []assetlibrary.Issue{}, Metadata: map[string]any{}}, nil
		}
	}
	return a.assets().Detail(id)
}

func (a *App) modelSources() ([]assetlibrary.PackPreviewSource, error) {
	a.modelIndexMu.Lock()
	sources := a.modelIndexSources
	epoch := a.modelIndexEpoch
	a.modelIndexMu.Unlock()
	if sources != nil {
		return sources, nil
	}
	sources, err := a.assets().PackPreviewSources()
	if err != nil {
		return nil, err
	}
	a.modelIndexMu.Lock()
	if epoch == a.modelIndexEpoch {
		a.modelIndexSources = sources
	}
	a.modelIndexMu.Unlock()
	return sources, nil
}

func (a *App) modelObjects(id string) ([]assetlibrary.Object, error) {
	a.modelIndexMu.Lock()
	objects, cached := a.modelIndex[id]
	epoch := a.modelIndexEpoch
	a.modelIndexMu.Unlock()
	if cached {
		return objects, nil
	}
	objects, err := a.assets().Objects(id)
	if err != nil {
		return nil, err
	}
	a.modelIndexMu.Lock()
	defer a.modelIndexMu.Unlock()
	if epoch == a.modelIndexEpoch {
		if a.modelIndex == nil {
			a.modelIndex = map[string][]assetlibrary.Object{}
		}
		a.modelIndex[id] = objects
	}
	return objects, nil
}

func (a *App) invalidateModelPicker() {
	a.modelIndexMu.Lock()
	a.modelIndexEpoch++
	a.modelIndex = nil
	a.modelIndexSources = nil
	a.modelIndexMu.Unlock()
	a.warmModelPicker()
}

func (a *App) warmModelPicker() {
	a.modelIndexMu.Lock()
	if a.modelIndexWarming {
		a.modelIndexMu.Unlock()
		return
	}
	a.modelIndexWarming = true
	a.modelIndexMu.Unlock()
	go func() {
		defer func() { a.modelIndexMu.Lock(); a.modelIndexWarming = false; a.modelIndexMu.Unlock() }()
		for attempt := 0; attempt < 8; attempt++ {
			time.Sleep(250 * time.Millisecond)
			a.modelIndexMu.Lock()
			epoch := a.modelIndexEpoch
			a.modelIndexMu.Unlock()
			a.packetMu.Lock()
			session := a.modelPickSession
			a.packetMu.Unlock()
			if session == "" {
				return
			}
			stack, err := a.commandExplorer().ResourcePacks(session)
			if err != nil {
				return
			}
			previews, err := a.modelSources()
			if err != nil {
				return
			}
			for i := len(stack) - 1; i >= 0; i-- {
				for _, preview := range previews {
					if preview.Source != nil && preview.Item.UUID == stack[i].UUID && preview.Item.Version == stack[i].Version {
						_, _ = a.modelObjects(preview.Item.ID)
					}
				}
			}
			a.modelIndexMu.Lock()
			unchanged := epoch == a.modelIndexEpoch
			a.modelIndexMu.Unlock()
			if unchanged {
				return
			}
		}
	}()
}
