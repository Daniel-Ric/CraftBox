package block

import (
	"github.com/df-mc/dragonfly/server/item"
)

type DiamondOre struct {
	solid
	bassDrum

	Type OreType
}

func (d DiamondOre) BreakInfo() BreakInfo {
	return newBreakInfo(d.Type.Hardness(), func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierIron.HarvestLevel
	}, pickaxeEffective, oreDrops(item.Diamond{}, d)).withXPDropRange(3, 7).withBlastResistance(3)
}

func (DiamondOre) SmeltInfo() item.SmeltInfo {
	return newOreSmeltInfo(item.NewStack(item.Diamond{}, 1), 1)
}

func (d DiamondOre) EncodeItem() (name string, meta int16) {
	return "minecraft:" + d.Type.Prefix() + "diamond_ore", 0
}

func (d DiamondOre) EncodeBlock() (string, map[string]any) {
	return "minecraft:" + d.Type.Prefix() + "diamond_ore", nil
}
