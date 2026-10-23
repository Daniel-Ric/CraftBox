package block

import (
	"github.com/df-mc/dragonfly/server/item"
)

type NetherGoldOre struct {
	solid
}

func (n NetherGoldOre) BreakInfo() BreakInfo {
	return newBreakInfo(3, pickaxeHarvestable, pickaxeEffective, multiOreDrops(item.GoldNugget{}, n, 2, 6)).withXPDropRange(0, 1)
}

func (NetherGoldOre) SmeltInfo() item.SmeltInfo {
	return newOreSmeltInfo(item.NewStack(item.GoldIngot{}, 1), 1)
}

func (NetherGoldOre) EncodeItem() (name string, meta int16) {
	return "minecraft:nether_gold_ore", 0
}

func (NetherGoldOre) EncodeBlock() (string, map[string]any) {
	return "minecraft:nether_gold_ore", nil
}
