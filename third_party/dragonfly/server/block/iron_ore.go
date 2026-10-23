package block

import (
	"github.com/df-mc/dragonfly/server/item"
)

type IronOre struct {
	solid
	bassDrum

	Type OreType
}

func (i IronOre) BreakInfo() BreakInfo {
	return newBreakInfo(i.Type.Hardness(), func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierStone.HarvestLevel
	}, pickaxeEffective, oreDrops(item.RawIron{}, i)).withBlastResistance(3)
}

func (IronOre) SmeltInfo() item.SmeltInfo {
	return newOreSmeltInfo(item.NewStack(item.IronIngot{}, 1), 0.7)
}

func (i IronOre) EncodeItem() (name string, meta int16) {
	return "minecraft:" + i.Type.Prefix() + "iron_ore", 0
}

func (i IronOre) EncodeBlock() (string, map[string]any) {
	return "minecraft:" + i.Type.Prefix() + "iron_ore", nil
}
