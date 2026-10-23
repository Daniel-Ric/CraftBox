package block

type Dripstone struct {
	solid
	bassDrum
}

func (d Dripstone) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(d)).withBlastResistance(1)
}

func (d Dripstone) EncodeItem() (name string, meta int16) {
	return "minecraft:dripstone_block", 0
}

func (d Dripstone) EncodeBlock() (string, map[string]any) {
	return "minecraft:dripstone_block", nil
}
