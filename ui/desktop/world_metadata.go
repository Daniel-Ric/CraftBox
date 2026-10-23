package desktop

import (
	"fmt"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"path/filepath"
	"strings"
)

func (a *App) EditWorld(request messages.EditWorld) error {
	a.editMu.Lock()
	defer a.editMu.Unlock()
	if request.WorldID == "" || (request.Name == nil && request.Spawn == nil) {
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
		if err := utils.ApplyWorldMetadata(folder, name, request.Spawn); err != nil {
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
