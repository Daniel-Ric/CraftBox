package block

type SulfurBricks struct {
	solid
	bassDrum
}

func (s SulfurBricks) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(s)).withBlastResistance(6)
}

func (SulfurBricks) EncodeItem() (name string, meta int16) {
	return "minecraft:sulfur_bricks", 0
}

func (SulfurBricks) EncodeBlock() (string, map[string]any) {
	return "minecraft:sulfur_bricks", nil
}
