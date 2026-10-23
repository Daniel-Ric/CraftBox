package block

import "github.com/df-mc/dragonfly/server/item"

type DeepslateBricks struct {
	solid
	bassDrum

	Cracked bool
}

func (d DeepslateBricks) BreakInfo() BreakInfo {
	return newBreakInfo(3.5, pickaxeHarvestable, pickaxeEffective, oneOf(d)).withBlastResistance(6)
}

func (d DeepslateBricks) SmeltInfo() item.SmeltInfo {
	if d.Cracked {
		return item.SmeltInfo{}
	}
	return newSmeltInfo(item.NewStack(DeepslateBricks{Cracked: true}, 1), 0.1)
}

func (d DeepslateBricks) EncodeItem() (name string, meta int16) {
	if d.Cracked {
		return "minecraft:cracked_deepslate_bricks", 0
	}
	return "minecraft:deepslate_bricks", 0
}

func (d DeepslateBricks) EncodeBlock() (string, map[string]any) {
	if d.Cracked {
		return "minecraft:cracked_deepslate_bricks", nil
	}
	return "minecraft:deepslate_bricks", nil
}
