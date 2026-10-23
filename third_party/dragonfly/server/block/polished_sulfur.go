package block

type PolishedSulfur struct {
	solid
	bassDrum
}

func (s PolishedSulfur) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(s)).withBlastResistance(6)
}

func (PolishedSulfur) EncodeItem() (name string, meta int16) {
	return "minecraft:polished_sulfur", 0
}

func (PolishedSulfur) EncodeBlock() (string, map[string]any) {
	return "minecraft:polished_sulfur", nil
}
