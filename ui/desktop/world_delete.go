package desktop

import (
	"errors"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"os"
	"path/filepath"
	"strings"
)

func deletedWorldEvent(event any) bool {
	var id string
	switch e := event.(type) {
	case *messages.EventWorldOpened:
		id = e.WorldID
	case *messages.EventSceneAvailable:
		id = e.WorldID
	case *messages.EventMapTiles:
		id = e.WorldID
	case *messages.EventPlayerPosition:
		id = e.WorldID
	case *messages.EventProcessingWorldUpdate:
		id = e.WorldID
	case *messages.EventFinishedSavingWorld:
		id = e.WorldID
	case *messages.EventResetMap:
		id = e.WorldID
	case *messages.EventWorldRegionCreated:
		id = e.Region.ID
	}
	return messages.WorldDeleted(id)
}

func captureDeleteFiles(w SavedWorld) ([]string, error) {
	folder := w.CaptureFolder
	if folder == "" && w.Path != "" {
		folder = strings.TrimSuffix(w.Path, ".mcworld")
	}
	if folder == "" {
		return nil, nil
	}
	root, err := filepath.Abs(utils.PathData("worlds"))
	if err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(folder)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(root, absolute)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || !strings.HasSuffix(filepath.Base(absolute), "--"+w.ID) || w.ID == "" {
		return nil, errors.New("World is outside the capture folder")
	}

	for p := absolute; p != root; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("Linked world folders cannot be deleted")
		}
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			return nil, err
		}
		rr, err := filepath.Rel(root, resolved)
		if err != nil || rr == ".." || strings.HasPrefix(rr, ".."+string(filepath.Separator)) {
			return nil, errors.New("World path leaves the capture folder")
		}
	}
	return []string{absolute, absolute + ".mcworld", absolute + ".png"}, nil
}

func (a *App) DeleteCapturedWorld(id string) error {
	a.mu.Lock()
	ids := map[string]bool{id: true}
	found := false
	for _, w := range a.state.Worlds {
		if w.ID == id {
			found = true
		}
	}
	if !found {
		a.mu.Unlock()
		return errors.New("Capture no longer exists")
	}
	for changed := true; changed; {
		changed = false
		for _, w := range a.state.Worlds {
			if ids[w.ParentID] && !ids[w.ID] {
				ids[w.ID] = true
				changed = true
			}
		}
	}
	var files []string
	if a.worldExportCancel != nil && ids[a.worldExportSource] {
		a.mu.Unlock()
		return errors.New("Cancel the world export before deleting its source")
	}
	var captures []SavedWorld
	for _, w := range a.state.Worlds {
		if ids[w.ID] {
			captures = append(captures, w)
			paths, err := captureDeleteFiles(w)
			if err != nil {
				a.mu.Unlock()
				return err
			}
			files = append(files, paths...)
		}
	}
	a.mu.Unlock()

	if err := messages.DiscardCapture(id); err != nil {
		return err
	}
	for child := range ids {
		if child != id {
			if err := messages.DiscardCapture(child); err != nil {
				return err
			}
		}
	}
	for _, capture := range captures {
		if authoritative := messages.CaptureFolder(capture.ID); authoritative != "" {
			capture.CaptureFolder = authoritative
			paths, err := captureDeleteFiles(capture)
			if err != nil {
				return err
			}
			files = append(files, paths...)
		}
	}
	for _, file := range files {
		if err := os.RemoveAll(file); err != nil {
			return fmt.Errorf("Delete capture: %w", err)
		}
	}
	a.mu.Lock()
	var stores []*scene.Store
	kept := a.state.Worlds[:0]
	for _, w := range a.state.Worlds {
		if !ids[w.ID] {
			kept = append(kept, w)
		}
	}
	a.state.Worlds = kept
	for removed := range ids {
		for key := range a.tileVersions {
			if strings.HasPrefix(key, removed+":") {
				delete(a.tileVersions, key)
			}
		}
		delete(a.worldTiles, removed)
		if store := a.scenes[removed]; store != nil {
			stores = append(stores, store)
			delete(a.scenes, removed)
		}
		delete(a.sceneResets, removed)
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
		emit("desktop:reset-map", nil)
		emit("desktop:tiles", a.GetMapTiles())
	}
	a.publish()
	return nil
}
