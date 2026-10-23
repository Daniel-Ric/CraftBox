package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"time"
)

type WoodFence struct {
	transparent
	bass
	sourceWaterDisplacer

	Wood WoodType

	Connections Connections
}

func (w WoodFence) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if connections := calculateConnections(w.Model().(connector), tx, pos); connections != w.Connections {
		w.Connections = connections
		tx.SetBlock(pos, w, nil)
	}
}

func (w WoodFence) DeriveState(pos cube.Pos, src world.BlockSource) world.Block {
	w.Connections = calculateConnections(w.Model().(connector), src, pos)
	return w
}

func (w WoodFence) BreakInfo() BreakInfo {
	return newBreakInfo(2, alwaysHarvestable, axeEffective, oneOf(w)).withBlastResistance(3)
}

func (WoodFence) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func (w WoodFence) FlammabilityInfo() FlammabilityInfo {
	if !w.Wood.Flammable() {
		return newFlammabilityInfo(0, 0, false)
	}
	return newFlammabilityInfo(5, 20, true)
}

func (w WoodFence) FuelInfo() item.FuelInfo {
	if !w.Wood.Flammable() {
		return item.FuelInfo{}
	}
	return newFuelInfo(time.Second * 15)
}

func (w WoodFence) EncodeBlock() (name string, properties map[string]any) {
	return "minecraft:" + w.Wood.String() + "_fence", w.Connections.properties()
}

func (w WoodFence) Model() world.BlockModel {
	return model.Fence{Wood: true}
}

func (w WoodFence) EncodeItem() (name string, meta int16) {
	return "minecraft:" + w.Wood.String() + "_fence", 0
}

func allFence() (fence []world.Block) {
	for _, w := range WoodTypes() {
		for _, c := range allConnections() {
			fence = append(fence, WoodFence{Wood: w, Connections: c})
		}
	}
	return
}
