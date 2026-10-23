package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
)

type NetherBrickFence struct {
	transparent
	sourceWaterDisplacer

	Connections Connections
}

func (n NetherBrickFence) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if connections := calculateConnections(n.Model().(connector), tx, pos); connections != n.Connections {
		n.Connections = connections
		tx.SetBlock(pos, n, nil)
	}
}

func (n NetherBrickFence) DeriveState(pos cube.Pos, src world.BlockSource) world.Block {
	n.Connections = calculateConnections(n.Model().(connector), src, pos)
	return n
}

func (n NetherBrickFence) BreakInfo() BreakInfo {
	return newBreakInfo(2, pickaxeHarvestable, pickaxeEffective, oneOf(n)).withBlastResistance(6)
}

func (NetherBrickFence) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func (n NetherBrickFence) Model() world.BlockModel {
	return model.Fence{}
}

func (NetherBrickFence) EncodeItem() (name string, meta int16) {
	return "minecraft:nether_brick_fence", 0
}

func (n NetherBrickFence) EncodeBlock() (string, map[string]any) {
	return "minecraft:nether_brick_fence", n.Connections.properties()
}

func allNetherBrickFence() (fence []world.Block) {
	for _, c := range allConnections() {
		fence = append(fence, NetherBrickFence{Connections: c})
	}
	return
}
