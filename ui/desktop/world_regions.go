package desktop

import (
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"math"
	"path/filepath"
)

func (a *App) updateRegionPlayersLocked(event *messages.EventPlayerPosition) {
	x, z := int32(math.Floor(float64(event.Position[0])/16)), int32(math.Floor(float64(event.Position[2])/16))
	for _, region := range a.worldRegions[event.WorldID] {
		if !region.Bounds.Contains(x, z) {
			continue
		}
		for i := range a.state.Worlds {
			if a.state.Worlds[i].ID == region.ID {
				copy(a.state.Worlds[i].Player[:], event.Position[:])
				a.state.Worlds[i].HasPlayer = true
			}
		}
	}
}

func (a *App) CreateWorldRegion(request messages.CreateWorldRegion) (messages.WorldRegion, error) {
	return messages.RequestWorldRegion(request)
}

func (a *App) storeRegionTileLocked(tile Tile, id string) {
	if messages.WorldDeleted(id) {
		return
	}
	tile.WorldID = id
	target := a.worldTiles[id]
	if target == nil {
		target = make(map[[2]int32]Tile)
		a.worldTiles[id] = target
	}
	target[[2]int32{tile.X, tile.Z}] = tile
	a.tileSequence++
	a.tileVersions[tileVersionKey(tile)] = a.tileSequence
}

func (a *App) addWorldRegion(region messages.WorldRegion) error {
	a.mu.Lock()
	if messages.WorldDeleted(region.ID) || messages.WorldDeleted(region.ParentID) {
		a.mu.Unlock()
		return nil
	}
	if a.worldRegions == nil {
		a.worldRegions = make(map[string][]messages.WorldRegion)
	}
	a.worldRegions[region.ParentID] = append(a.worldRegions[region.ParentID], region)
	if store := a.scenes[region.ParentID]; store != nil {
		if a.sceneResets == nil {
			a.sceneResets = make(map[string]uint64)
		}
		a.sceneResets[region.ParentID] = store.Invalidate()
	}
	if region.Move {
		for i := range a.state.Worlds {
			item := &a.state.Worlds[i]
			if item.ID == region.ParentID && item.Spawn != nil && region.Bounds.Contains((*item.Spawn)[0]>>4, (*item.Spawn)[2]>>4) {
				item.Spawn = nil
			}
		}
	}
	a.state.Worlds = append(a.state.Worlds, SavedWorld{ID: region.ID, ParentID: region.ParentID, Name: region.Name, State: "Capturing"})
	for _, parent := range a.state.Worlds {
		if parent.ID == region.ParentID {
			a.state.Worlds[len(a.state.Worlds)-1].CapturePoints = messages.CapturePointsInBounds(parent.CapturePoints, region.Bounds)
			if parent.CaptureFolder != "" {
				a.state.Worlds[len(a.state.Worlds)-1].CaptureFolder = filepath.Join(filepath.Dir(parent.CaptureFolder), utils.MakeValidFilename(region.Name)+"--"+region.ID)
			}
			a.state.Worlds[len(a.state.Worlds)-1].HeightRange = parent.HeightRange
			break
		}
	}
	if region.Bounds.Contains(int32(math.Floor(float64(a.state.Player[0])/16)), int32(math.Floor(float64(a.state.Player[2])/16))) {
		last := len(a.state.Worlds) - 1
		a.state.Worlds[last].Player = a.state.Player
		a.state.Worlds[last].HasPlayer = a.state.HasPlayer
	}
	for key, tile := range a.worldTiles[region.ParentID] {
		if !region.Bounds.Contains(tile.X, tile.Z) {
			continue
		}
		a.storeRegionTileLocked(tile, region.ID)
		if region.Move {
			delete(a.worldTiles[region.ParentID], key)
		}
	}
	a.tileSequence++
	a.tileResetSequence = a.tileSequence
	emit := a.emit
	a.mu.Unlock()
	if emit != nil {
		emit("desktop:reset-map", nil)
		emit("desktop:tiles", a.GetMapTiles())
	}
	a.publish()
	return nil
}
