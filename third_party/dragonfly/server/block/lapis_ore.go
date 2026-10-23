package block

import (
	"github.com/df-mc/dragonfly/server/item"
)

type LapisOre struct {
	solid
	bassDrum

	Type OreType
}

func (l LapisOre) BreakInfo() BreakInfo {
	return newBreakInfo(l.Type.Hardness(), func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierStone.HarvestLevel
	}, pickaxeEffective, multiOreDrops(item.LapisLazuli{}, l, 4, 9)).withXPDropRange(2, 5).withBlastResistance(3)
}

func (LapisOre) SmeltInfo() item.SmeltInfo {
	return newOreSmeltInfo(item.NewStack(item.LapisLazuli{}, 1), 0.2)
}

func (l LapisOre) EncodeItem() (name string, meta int16) {
	return "minecraft:" + l.Type.Prefix() + "lapis_ore", 0
}

func (l LapisOre) EncodeBlock() (string, map[string]any) {
	return "minecraft:" + l.Type.Prefix() + "lapis_ore", nil
}
