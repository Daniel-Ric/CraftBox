package worlds

import (
	"fmt"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/df-mc/dragonfly/server/block/cube"
)

func (w *worldsHandler) captureContainerInventory(state *worldstate.World, container *itemContainer) error {
	if container == nil || container.OpenPacket == nil || container.Content == nil {
		return nil
	}

	if id := container.OpenPacket.ContainerEntityUniqueID; id != 0 && id != -1 {
		return nil
	}
	items, err := utils.CapturedInventoryNBT(w.serverState.blocks, container.Content.Content, w.serverState.itemNames, w.serverState.useHashedRids)
	if err != nil {
		return err
	}
	p := container.OpenPacket.ContainerPosition
	return state.SetBlockNBT(cube.Pos{int(p.X()), int(p.Y()), int(p.Z())}, map[string]any{"Items": items}, true)
}
func (w *worldsHandler) flushContainerInventories(state *worldstate.World) error {
	for id, container := range w.serverState.openItemContainers {
		if err := w.captureContainerInventory(state, container); err != nil {
			return fmt.Errorf("capture open container %d: %w", id, err)
		}
	}
	return nil
}
