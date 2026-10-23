package block

type Cinnabar struct {
	solid
	bassDrum

	Chiseled bool
}

func (c Cinnabar) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(c)).withBlastResistance(6)
}

func (c Cinnabar) EncodeItem() (name string, meta int16) {
	if c.Chiseled {
		return "minecraft:chiseled_cinnabar", 0
	}
	return "minecraft:cinnabar", 0
}

func (c Cinnabar) EncodeBlock() (string, map[string]any) {
	if c.Chiseled {
		return "minecraft:chiseled_cinnabar", nil
	}
	return "minecraft:cinnabar", nil
}
