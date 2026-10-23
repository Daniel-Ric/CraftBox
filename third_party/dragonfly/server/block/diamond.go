package block

import (
	"github.com/df-mc/dragonfly/server/item"
)

type Diamond struct {
	solid
}

func (d Diamond) BreakInfo() BreakInfo {
	return newBreakInfo(5, func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierIron.HarvestLevel
	}, pickaxeEffective, oneOf(d)).withBlastResistance(6)
}

func (Diamond) PowersBeacon() bool {
	return true
}

func (Diamond) EncodeItem() (name string, meta int16) {
	return "minecraft:diamond_block", 0
}

func (Diamond) EncodeBlock() (string, map[string]any) {
	return "minecraft:diamond_block", nil
}
