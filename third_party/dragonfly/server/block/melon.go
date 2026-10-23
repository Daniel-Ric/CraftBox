package block

import (
	"github.com/df-mc/dragonfly/server/item"
)

type Melon struct {
	solid
}

func (m Melon) BreakInfo() BreakInfo {
	return newBreakInfo(1, alwaysHarvestable, axeEffective, discreteDrops(item.MelonSlice{}, m, 3, 7, 9))
}

func (Melon) CompostChance() float64 {
	return 0.65
}

func (Melon) EncodeItem() (name string, meta int16) {
	return "minecraft:melon_block", 0
}

func (Melon) EncodeBlock() (string, map[string]any) {
	return "minecraft:melon_block", nil
}
