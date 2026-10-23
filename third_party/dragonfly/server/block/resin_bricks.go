package block

type ResinBricks struct {
	solid
	bassDrum

	Chiseled bool
}

func (r ResinBricks) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(r)).withBlastResistance(6)
}

func (r ResinBricks) EncodeItem() (name string, meta int16) {
	if r.Chiseled {
		return "minecraft:chiseled_resin_bricks", 0
	}
	return "minecraft:resin_bricks", 0
}

func (r ResinBricks) EncodeBlock() (string, map[string]any) {
	if r.Chiseled {
		return "minecraft:chiseled_resin_bricks", nil
	}
	return "minecraft:resin_bricks", nil
}
