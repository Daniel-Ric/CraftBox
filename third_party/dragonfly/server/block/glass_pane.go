package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type GlassPane struct {
	transparent
	thin
	clicksAndSticks
	sourceWaterDisplacer

	Connections Connections
}

func (p GlassPane) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if connections := calculateConnections(p.Model().(connector), tx, pos); connections != p.Connections {
		p.Connections = connections
		tx.SetBlock(pos, p, nil)
	}
}

func (p GlassPane) DeriveState(pos cube.Pos, src world.BlockSource) world.Block {
	p.Connections = calculateConnections(p.Model().(connector), src, pos)
	return p
}

func (p GlassPane) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func (p GlassPane) BreakInfo() BreakInfo {
	return newBreakInfo(0.3, alwaysHarvestable, nothingEffective, silkTouchOnlyDrop(p))
}

func (GlassPane) EncodeItem() (name string, meta int16) {
	return "minecraft:glass_pane", meta
}

func (p GlassPane) EncodeBlock() (string, map[string]any) {
	return "minecraft:glass_pane", p.Connections.properties()
}

func allGlassPane() (panes []world.Block) {
	for _, c := range allConnections() {
		panes = append(panes, GlassPane{Connections: c})
	}
	return
}
