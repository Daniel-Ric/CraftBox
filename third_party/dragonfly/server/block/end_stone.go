package block

type EndStone struct {
	solid
	bassDrum
}

func (e EndStone) BreakInfo() BreakInfo {
	return newBreakInfo(3, pickaxeHarvestable, pickaxeEffective, oneOf(e)).withBlastResistance(9)
}

func (EndStone) EncodeItem() (name string, meta int16) {
	return "minecraft:end_stone", 0
}

func (EndStone) EncodeBlock() (string, map[string]any) {
	return "minecraft:end_stone", nil
}
