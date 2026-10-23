package block

type PolishedTuff struct {
	solid
	bassDrum
}

func (t PolishedTuff) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(t)).withBlastResistance(6)
}

func (t PolishedTuff) EncodeItem() (name string, meta int16) {
	return "minecraft:polished_tuff", 0
}

func (t PolishedTuff) EncodeBlock() (string, map[string]any) {
	return "minecraft:polished_tuff", nil
}
