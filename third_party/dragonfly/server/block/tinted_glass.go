package block

type TintedGlass struct {
	solid
	clicksAndSticks
}

func (TintedGlass) PreventsSuffocation() bool {
	return true
}

func (g TintedGlass) BreakInfo() BreakInfo {
	return newBreakInfo(0.3, alwaysHarvestable, nothingEffective, oneOf(g)).withBlastResistance(0.3)
}

func (TintedGlass) EncodeItem() (name string, meta int16) {
	return "minecraft:tinted_glass", 0
}

func (TintedGlass) EncodeBlock() (string, map[string]any) {
	return "minecraft:tinted_glass", nil
}
