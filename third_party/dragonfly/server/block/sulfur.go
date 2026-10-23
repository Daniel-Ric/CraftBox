package block

type Sulfur struct {
	solid
	bassDrum

	Chiseled bool
}

func (s Sulfur) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(s)).withBlastResistance(6)
}

func (s Sulfur) EncodeItem() (name string, meta int16) {
	if s.Chiseled {
		return "minecraft:chiseled_sulfur", 0
	}
	return "minecraft:sulfur", 0
}

func (s Sulfur) EncodeBlock() (string, map[string]any) {
	if s.Chiseled {
		return "minecraft:chiseled_sulfur", nil
	}
	return "minecraft:sulfur", nil
}
