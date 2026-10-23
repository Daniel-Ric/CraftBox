package block

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world/sound"
)

type Iron struct {
	solid
}

func (i Iron) Instrument() sound.Instrument {
	return sound.IronXylophone()
}

func (i Iron) BreakInfo() BreakInfo {
	return newBreakInfo(5, func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierStone.HarvestLevel
	}, pickaxeEffective, oneOf(i)).withBlastResistance(6)
}

func (Iron) PowersBeacon() bool {
	return true
}

func (Iron) EncodeItem() (name string, meta int16) {
	return "minecraft:iron_block", 0
}

func (Iron) EncodeBlock() (string, map[string]any) {
	return "minecraft:iron_block", nil
}
