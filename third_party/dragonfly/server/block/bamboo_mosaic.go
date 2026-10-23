package block

import (
	"github.com/df-mc/dragonfly/server/item"
	"time"
)

type BambooMosaic struct {
	solid
	bass
}

func (BambooMosaic) FlammabilityInfo() FlammabilityInfo {
	return newFlammabilityInfo(5, 20, true)
}

func (b BambooMosaic) BreakInfo() BreakInfo {
	return newBreakInfo(2, alwaysHarvestable, axeEffective, oneOf(b)).withBlastResistance(3)
}

func (BambooMosaic) RepairsWoodTools() bool {
	return true
}

func (BambooMosaic) FuelInfo() item.FuelInfo {
	return newFuelInfo(time.Second * 15)
}

func (BambooMosaic) EncodeItem() (name string, meta int16) {
	return "minecraft:bamboo_mosaic", 0
}

func (BambooMosaic) EncodeBlock() (string, map[string]any) {
	return "minecraft:bamboo_mosaic", nil
}
