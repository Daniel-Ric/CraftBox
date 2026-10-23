package block

type PolishedCinnabar struct {
	solid
	bassDrum
}

func (c PolishedCinnabar) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(c)).withBlastResistance(6)
}

func (PolishedCinnabar) EncodeItem() (name string, meta int16) {
	return "minecraft:polished_cinnabar", 0
}

func (PolishedCinnabar) EncodeBlock() (string, map[string]any) {
	return "minecraft:polished_cinnabar", nil
}
