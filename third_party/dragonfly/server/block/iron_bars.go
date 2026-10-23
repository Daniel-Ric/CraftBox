package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type IronBars struct {
	transparent
	thin
	sourceWaterDisplacer

	Connections Connections
}

func (i IronBars) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if connections := calculateConnections(i.Model().(connector), tx, pos); connections != i.Connections {
		i.Connections = connections
		tx.SetBlock(pos, i, nil)
	}
}

func (i IronBars) DeriveState(pos cube.Pos, src world.BlockSource) world.Block {
	i.Connections = calculateConnections(i.Model().(connector), src, pos)
	return i
}

func (i IronBars) BreakInfo() BreakInfo {
	return newBreakInfo(5, pickaxeHarvestable, pickaxeEffective, oneOf(i)).withBlastResistance(6)
}

func (i IronBars) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func (IronBars) EncodeItem() (name string, meta int16) {
	return "minecraft:iron_bars", 0
}

func (i IronBars) EncodeBlock() (string, map[string]any) {
	return "minecraft:iron_bars", i.Connections.properties()
}

func allIronBars() (bars []world.Block) {
	for _, c := range allConnections() {
		bars = append(bars, IronBars{Connections: c})
	}
	return
}
