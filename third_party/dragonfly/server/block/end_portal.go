package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/portal"
)

type EndPortal struct {
	transparent
}

func (EndPortal) Model() world.BlockModel {
	return model.Empty{}
}

func (EndPortal) LightEmissionLevel() uint8 {
	return 15
}

func (EndPortal) HasLiquidDrops() bool {
	return false
}

func (EndPortal) Portal() world.Dimension {
	return world.End
}

func (EndPortal) EncodeNBT() map[string]any {
	return map[string]any{"id": "EndPortal"}
}

func (e EndPortal) DecodeNBT(map[string]any) any {
	return e
}

func (EndPortal) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if portal.EndPortalRingIntact(tx, pos) {
		return
	}
	portal.DeactivateEndPortal(tx, pos)
}

func (EndPortal) EntityInside(_ cube.Pos, tx *world.Tx, e world.Entity) {
	if t, ok := e.(portalTraveller); ok {
		t.TravelThroughPortal(tx, world.End)
	}
}

func (EndPortal) EncodeBlock() (string, map[string]any) {
	return "minecraft:end_portal", nil
}
