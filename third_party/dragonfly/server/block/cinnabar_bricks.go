package block

type CinnabarBricks struct {
	solid
	bassDrum
}

func (c CinnabarBricks) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(c)).withBlastResistance(6)
}

func (CinnabarBricks) EncodeItem() (name string, meta int16) {
	return "minecraft:cinnabar_bricks", 0
}

func (CinnabarBricks) EncodeBlock() (string, map[string]any) {
	return "minecraft:cinnabar_bricks", nil
}
