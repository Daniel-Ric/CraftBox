package block

type Bricks struct {
	solid
	bassDrum
}

func (b Bricks) BreakInfo() BreakInfo {
	return newBreakInfo(2, pickaxeHarvestable, pickaxeEffective, oneOf(b)).withBlastResistance(6)
}

func (Bricks) EncodeItem() (name string, meta int16) {
	return "minecraft:brick_block", 0
}

func (Bricks) EncodeBlock() (string, map[string]any) {
	return "minecraft:brick_block", nil
}
