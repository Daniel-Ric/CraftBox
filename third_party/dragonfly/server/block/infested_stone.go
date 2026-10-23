package block

type InfestedStone struct {
	solid
	flute
}

func (i InfestedStone) BreakInfo() BreakInfo {
	return newBreakInfo(0.75, pickaxeHarvestable, pickaxeEffective, silkTouchOnlyDrop(i)).withBlastResistance(0.75)
}

func (InfestedStone) EncodeItem() (name string, meta int16) {
	return "minecraft:infested_stone", 0
}

func (InfestedStone) EncodeBlock() (string, map[string]any) {
	return "minecraft:infested_stone", nil
}
