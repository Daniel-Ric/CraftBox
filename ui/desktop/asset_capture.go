package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"weak"

	"github.com/Daniel-Ric/CraftBox/internal/assetlibrary"
	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

func (a *App) assets() *assetlibrary.Library {
	a.assetMu.Lock()
	defer a.assetMu.Unlock()
	if a.assetLibrary == nil {
		a.assetLibrary = assetlibrary.New(utils.PathData("asset-library"))
	}
	return a.assetLibrary
}
func (a *App) enqueueAsset(task func() error, size int) {
	a.assetMu.Lock()
	if size > 16<<20 || a.assetQueuedBytes+size > 48<<20 {
		a.assetMu.Unlock()
		diagnostics.Record("warning", "assets", "library.capture.memory", "Asset library queue is full; captured originals remain on disk", nil)
		return
	}
	if a.assetTasks == nil {
		a.assetTasks = make(chan func() error, 128)
		a.assetWorkerDone = make(chan struct{})
		done := a.assetWorkerDone
		queue := a.assetTasks
		go func() {
			defer close(done)
			for job := range queue {
				if e := job(); e != nil {
					diagnostics.Record("warning", "assets", "library.capture.failed", "Asset could not be added to the library", map[string]any{"error": e.Error()})
				}
				a.mu.Lock()
				emit := a.emit
				a.mu.Unlock()
				if emit != nil {
					emit("desktop:assets", true)
				}
			}
		}()
	}
	queue := a.assetTasks
	a.assetQueuedBytes += size
	select {
	case queue <- func() error {
		defer func() { a.assetMu.Lock(); a.assetQueuedBytes -= size; a.assetMu.Unlock() }()
		return task()
	}:
	default:
		a.assetQueuedBytes -= size
		diagnostics.Record("warning", "assets", "library.capture.busy", "Asset library is busy; captured originals remain on disk", nil)
	}
	a.assetMu.Unlock()
}
func (a *App) flushAssets() {
	a.assetMu.Lock()
	queue, done := a.assetTasks, a.assetWorkerDone
	if queue != nil {
		close(queue)
		a.assetTasks = nil
	}
	a.assetMu.Unlock()
	if done != nil {
		<-done
	}
}
func (a *App) persistAssetEvent(event any) {
	a.packetMu.Lock()
	recorder := a.packetRecorder
	a.packetMu.Unlock()
	if recorder != nil {
		settings := recorder.Settings()
		a.packetMu.Lock()
		if a.packetModulesSet {
			settings = a.packetModules
		}
		a.packetMu.Unlock()
		a.mu.Lock()
		command := a.state.Command
		a.mu.Unlock()
		switch event.(type) {
		case *messages.EventResourcePackAvailable, *messages.EventCapturedBlockDefinitions:
			if !settings.CapturePacks && command != "packs" {
				return
			}
		case *messages.EventPlayerSkin, *messages.EventPlayerEquipment:
			if !settings.CaptureSkins && command != "skins" {
				return
			}
		}
	}
	switch e := event.(type) {
	case *messages.EventPlayerSkin:
		raw, err := json.Marshal(e.Skin)
		if err != nil {
			return
		}
		name := e.PlayerName
		playerUUID, verified := e.PlayerUUID, e.XUID != ""
		a.enqueueAsset(func() error {
			var skin protocol.Skin
			if err := json.Unmarshal(raw, &skin); err != nil {
				return err
			}
			_, err := a.assets().CaptureSkin(context.Background(), name, "Capture", skin, map[string]any{"playerUUID": playerUUID, "verifiedPlayer": verified})
			return err
		}, len(raw))
	case *messages.EventPlayerEquipment:
		slots := map[string]string{}
		for key, value := range e.Equipment {
			slots[key] = value
		}
		playerUUID, name := e.PlayerUUID, e.PlayerName
		a.enqueueAsset(func() error { return a.assets().SetCapturedEquipment(playerUUID, name, slots) }, 1024)
	case *messages.EventResourcePackAvailable:
		if e.Pack == nil {
			return
		}
		pack := e.Pack
		a.enqueueAsset(func() error {
			id, err := a.captureAssetPack(pack)
			if err != nil {
				return err
			}
			a.assetMu.Lock()
			if len(a.assetCapturedPacks) >= 128 {
				a.assetCapturedPacks = map[weak.Pointer[resource.Pack]]string{}
			}
			if a.assetCapturedPacks == nil {
				a.assetCapturedPacks = map[weak.Pointer[resource.Pack]]string{}
			}
			a.assetCapturedPacks[weak.Make(pack)] = id
			a.assetMu.Unlock()
			return nil
		}, 256)
	case *messages.EventCapturedBlockDefinitions:
		var size int
		docs := make([]json.RawMessage, len(e.Definitions))
		for i, raw := range e.Definitions {
			size += len(raw)
			docs[i] = raw
		}
		packs := append([]*resource.Pack{}, e.Packs...)
		a.enqueueAsset(func() error {
			var first error
			for _, pack := range packs {
				if pack == nil {
					continue
				}
				a.assetMu.Lock()
				id := a.assetCapturedPacks[weak.Make(pack)]
				delete(a.assetCapturedPacks, weak.Make(pack))
				a.assetMu.Unlock()
				if id == "" {
					var err error
					id, err = a.captureAssetPack(pack)
					if err != nil {
						if first == nil {
							first = err
						}
						continue
					}
				}
				if err := a.assets().SaveBlockDefinitions(id, docs); err != nil && first == nil {
					first = err
				}
			}
			return first
		}, size)
	}
}
func (a *App) captureAssetPack(pack *resource.Pack) (string, error) {
	readable, err := utils.ReadablePack(pack)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "craftbox-readable-pack-*.mcpack")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	_, err = readable.WriteTo(f)
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	items, err := a.assets().Import(context.Background(), "packs", []string{f.Name()}, "Capture")
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", fmt.Errorf("Captured pack did not contain an asset")
	}
	return items[0].ID, nil
}
func (a *App) assetInstallation(id string) (assetlibrary.Installation, error) {
	all, err := assetlibrary.InstalledMinecraft()
	for _, v := range all {
		if v.ID == id {
			return v, nil
		}
	}
	if err != nil {
		return assetlibrary.Installation{}, fmt.Errorf("Minecraft detection failed: %w", err)
	}
	return assetlibrary.Installation{}, fmt.Errorf("This Minecraft installation is no longer available. Refresh the list")
}
