package block

import "github.com/df-mc/dragonfly/server/item"

type DeepslateTiles struct {
	solid
	bassDrum

	Cracked bool
}

func (d DeepslateTiles) BreakInfo() BreakInfo {
	return newBreakInfo(3.5, pickaxeHarvestable, pickaxeEffective, oneOf(d)).withBlastResistance(6)
}

func (d DeepslateTiles) SmeltInfo() item.SmeltInfo {
	if d.Cracked {
		return item.SmeltInfo{}
	}
	return newSmeltInfo(item.NewStack(DeepslateTiles{Cracked: true}, 1), 0.1)
}

func (d DeepslateTiles) EncodeItem() (name string, meta int16) {
	if d.Cracked {
		return "minecraft:cracked_deepslate_tiles", 0
	}
	return "minecraft:deepslate_tiles", 0
}

func (d DeepslateTiles) EncodeBlock() (string, map[string]any) {
	if d.Cracked {
		return "minecraft:cracked_deepslate_tiles", nil
	}
	return "minecraft:deepslate_tiles", nil
}
