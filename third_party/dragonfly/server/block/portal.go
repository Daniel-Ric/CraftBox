package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/portal"
)

type Portal struct {
	transparent

	Axis cube.Axis
}

type portalTraveller interface {
	TravelThroughPortal(tx *world.Tx, target world.Dimension)
}

func (p Portal) Model() world.BlockModel {
	return model.Portal{Axis: p.Axis}
}

func (Portal) Portal() world.Dimension {
	return world.Nether
}

func (Portal) LightEmissionLevel() uint8 {
	return 11
}

func (p Portal) HasLiquidDrops() bool {
	return false
}

func (p Portal) EncodeBlock() (string, map[string]any) {
	return "minecraft:portal", map[string]any{"portal_axis": p.Axis.String()}
}

func (p Portal) NeighbourUpdateTick(pos, neighbour cube.Pos, tx *world.Tx) {
	face, ok := pos.NeighbourFace(neighbour)
	if !ok {
		return
	}
	axis := face.Axis()
	if axis != cube.Y && axis != p.Axis {
		return
	}
	if n, ok := portal.NetherPortalFromPos(tx, pos); ok && n.Framed() && n.Activated() {
		return
	}
	portal.DeactivateNetherPortal(tx, pos)
}

func (p Portal) EntityInside(_ cube.Pos, tx *world.Tx, e world.Entity) {
	if t, ok := e.(portalTraveller); ok {
		t.TravelThroughPortal(tx, p.Portal())
	}
}
