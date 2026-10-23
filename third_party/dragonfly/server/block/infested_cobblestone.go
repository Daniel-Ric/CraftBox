package block

type InfestedCobblestone struct {
	solid
	flute
}

func (i InfestedCobblestone) BreakInfo() BreakInfo {
	return newBreakInfo(1, pickaxeHarvestable, pickaxeEffective, silkTouchOnlyDrop(i)).withBlastResistance(0.75)
}

func (InfestedCobblestone) EncodeItem() (name string, meta int16) {
	return "minecraft:infested_cobblestone", 0
}

func (InfestedCobblestone) EncodeBlock() (string, map[string]any) {
	return "minecraft:infested_cobblestone", nil
}
