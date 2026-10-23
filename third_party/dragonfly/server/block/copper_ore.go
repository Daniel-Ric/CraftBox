package block

import (
	"github.com/df-mc/dragonfly/server/item"
)

type CopperOre struct {
	solid
	bassDrum

	Type OreType
}

func (c CopperOre) BreakInfo() BreakInfo {
	return newBreakInfo(c.Type.Hardness(), func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierStone.HarvestLevel
	}, pickaxeEffective, multiOreDrops(item.RawCopper{}, c, 2, 5)).withBlastResistance(3)
}

func (CopperOre) SmeltInfo() item.SmeltInfo {
	return newOreSmeltInfo(item.NewStack(item.CopperIngot{}, 1), 0.7)
}

func (c CopperOre) EncodeItem() (name string, meta int16) {
	return "minecraft:" + c.Type.Prefix() + "copper_ore", 0
}

func (c CopperOre) EncodeBlock() (string, map[string]any) {
	return "minecraft:" + c.Type.Prefix() + "copper_ore", nil
}
