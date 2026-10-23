package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

type StainedGlassPane struct {
	transparent
	thin
	clicksAndSticks
	sourceWaterDisplacer

	Colour item.Colour

	Connections Connections
}

func (p StainedGlassPane) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if connections := calculateConnections(p.Model().(connector), tx, pos); connections != p.Connections {
		p.Connections = connections
		tx.SetBlock(pos, p, nil)
	}
}

func (p StainedGlassPane) DeriveState(pos cube.Pos, src world.BlockSource) world.Block {
	p.Connections = calculateConnections(p.Model().(connector), src, pos)
	return p
}

func (p StainedGlassPane) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func (p StainedGlassPane) BreakInfo() BreakInfo {
	return newBreakInfo(0.3, alwaysHarvestable, nothingEffective, silkTouchOnlyDrop(p))
}

func (p StainedGlassPane) EncodeItem() (name string, meta int16) {
	return "minecraft:" + p.Colour.String() + "_stained_glass_pane", 0
}

func (p StainedGlassPane) EncodeBlock() (name string, properties map[string]any) {
	return "minecraft:" + p.Colour.String() + "_stained_glass_pane", p.Connections.properties()
}

func allStainedGlassPane() []world.Block {
	b := make([]world.Block, 0, 16)
	for _, colour := range item.Colours() {
		for _, c := range allConnections() {
			b = append(b, StainedGlassPane{Colour: colour, Connections: c})
		}
	}
	return b
}
