package desktop

import (
	"context"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/worldexport"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"path/filepath"
	"strings"
)

type WorldExportConfiguration struct {
	WorldID     string                `json:"worldId"`
	Name        string                `json:"name"`
	Capturing   bool                  `json:"capturing"`
	HeightRange [2]int                `json:"heightRange"`
	Profiles    []worldexport.Profile `json:"profiles"`
	Options     worldexport.Options   `json:"options"`
}

func (a *App) GetWorldExportConfig(id string) (WorldExportConfiguration, error) {
	a.mu.Lock()
	var item SavedWorld
	found := false
	for _, w := range a.state.Worlds {
		if w.ID == id {
			item = w
			found = true
			break
		}
	}
	a.mu.Unlock()
	if !found {
		return WorldExportConfiguration{}, fmt.Errorf("World not found")
	}
	options := worldexport.Options{Profile: "original"}
	if stored := messages.WorldMetadata(id).Export; stored != nil {
		options = *stored
	} else if item.State == "finished" {
		folder := strings.TrimSuffix(item.Path, filepath.Ext(item.Path))
		var err error
		options, err = worldexport.ReadOptions(folder)
		if err != nil {
			return WorldExportConfiguration{}, err
		}
	}
	return WorldExportConfiguration{WorldID: id, Name: item.Name, Capturing: item.State != "finished", HeightRange: item.HeightRange, Profiles: worldexport.Profiles(), Options: options}, nil
}

func (a *App) EditWorld(request messages.EditWorld) error {
	a.editMu.Lock()
	defer a.editMu.Unlock()
	if request.Export != nil {
		if len(request.Export.Dimensions) > 0 || len(request.Export.Biomes) > 0 || request.Export.PlayerDimension != nil || request.Export.Compact {
			return fmt.Errorf("Use Export world copy for dimension, region, biome and storage transformations")
		}
		if err := request.Export.Validate(); err != nil {
			return err
		}
		if request.Name == nil {
			request.Name = request.Export.Settings.Name
		}
		if request.Spawn == nil {
			request.Spawn = request.Export.Settings.Spawn
		}
	}
	if request.WorldID == "" || (request.Name == nil && request.Spawn == nil && request.Export == nil) {
		return fmt.Errorf("World and change are required")
	}
	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" || len([]rune(name)) > 80 {
			return fmt.Errorf("Name must contain 1 to 80 characters")
		}
		request.Name = &name
	}
	a.mu.Lock()
	index := -1
	for i, item := range a.state.Worlds {
		if item.ID == request.WorldID {
			index = i
			break
		}
	}
	if index < 0 {
		a.mu.Unlock()
		return fmt.Errorf("World not found")
	}
	item := a.state.Worlds[index]
	if request.Spawn != nil {
		pos := *request.Spawn
		height := item.HeightRange
		if height[0] == height[1] {
			height = [2]int{-64, 319}
		}
		if int(pos[1]) < height[0] || int(pos[1])+1 > height[1] {
			a.mu.Unlock()
			return fmt.Errorf("Spawn point is outside the dimension height (%d to %d)", height[0], height[1]-1)
		}
		if _, ok := a.worldTiles[item.ID][[2]int32{pos[0] >> 4, pos[2] >> 4}]; !ok {
			a.mu.Unlock()
			return fmt.Errorf("Select a spawn point within this world's captured terrain")
		}
	}

	a.mu.Unlock()
	if item.State == "finished" {
		folder := strings.TrimSuffix(item.Path, filepath.Ext(item.Path))
		name := ""
		if request.Name != nil {
			name = *request.Name
		}
		options := worldexport.Options{}
		if request.Export != nil {
			options = *request.Export
		}
		if name != "" {
			options.Settings.Name = &name
		}
		options.Settings.Spawn = request.Spawn
		if err := worldexport.UpdateMetadata(context.Background(), folder, options); err != nil {
			return err
		}
		if err := utils.ReplaceWorldArchive(folder, item.Path); err != nil {
			return err
		}
		messages.StoreWorldMetadata(request)
	} else {
		parent := item.ParentID
		if parent == "" {
			parent = item.ID
		}
		if err := messages.RequestWorldMetadata(parent, request); err != nil {
			return err
		}
	}
	a.mu.Lock()
	index = -1
	for i, world := range a.state.Worlds {
		if world.ID == request.WorldID {
			index = i
			break
		}
	}
	if index >= 0 && request.Name != nil {
		a.state.Worlds[index].Name = *request.Name
	}
	if index >= 0 && request.Spawn != nil {
		a.state.Worlds[index].Spawn = messages.WorldMetadata(request.WorldID).Spawn
	}
	a.mu.Unlock()
	a.publish()
	return nil
}
