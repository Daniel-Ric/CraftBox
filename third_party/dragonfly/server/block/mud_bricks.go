package block

type MudBricks struct {
	solid
	bassDrum
}

func (m MudBricks) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, alwaysHarvestable, nothingEffective, oneOf(m)).withBlastResistance(3)
}

func (MudBricks) EncodeItem() (name string, meta int16) {
	return "minecraft:mud_bricks", 0
}

func (MudBricks) EncodeBlock() (string, map[string]any) {
	return "minecraft:mud_bricks", nil
}
