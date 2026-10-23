package block

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"time"
)

type Planks struct {
	solid
	bass

	Wood WoodType
}

func (p Planks) FlammabilityInfo() FlammabilityInfo {
	if !p.Wood.Flammable() {
		return newFlammabilityInfo(0, 0, false)
	}
	return newFlammabilityInfo(5, 20, true)
}

func (p Planks) BreakInfo() BreakInfo {
	return newBreakInfo(2, alwaysHarvestable, axeEffective, oneOf(p)).withBlastResistance(3)
}

func (p Planks) RepairsWoodTools() bool {
	return true
}

func (p Planks) FuelInfo() item.FuelInfo {
	if !p.Wood.Flammable() {
		return item.FuelInfo{}
	}
	return newFuelInfo(time.Second * 15)
}

func (p Planks) EncodeItem() (name string, meta int16) {
	return "minecraft:" + p.Wood.String() + "_planks", 0
}

func (p Planks) EncodeBlock() (name string, properties map[string]any) {
	return "minecraft:" + p.Wood.String() + "_planks", nil
}

func allPlanks() (planks []world.Block) {
	for _, w := range WoodTypes() {
		planks = append(planks, Planks{Wood: w})
	}
	return
}
