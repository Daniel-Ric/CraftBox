package block

type SmoothBasalt struct {
	solid
	bassDrum
}

func (SmoothBasalt) EncodeBlock() (string, map[string]any) {
	return "minecraft:smooth_basalt", nil
}

func (SmoothBasalt) EncodeItem() (name string, meta int16) {
	return "minecraft:smooth_basalt", 0
}

func (s SmoothBasalt) BreakInfo() BreakInfo {
	return newBreakInfo(1.25, pickaxeHarvestable, pickaxeEffective, oneOf(s)).withBlastResistance(4.2)
}
