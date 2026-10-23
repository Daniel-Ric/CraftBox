package block

type NetherWartBlock struct {
	solid

	Warped bool
}

func (n NetherWartBlock) BreakInfo() BreakInfo {
	return newBreakInfo(1, alwaysHarvestable, hoeEffective, oneOf(n))
}

func (NetherWartBlock) CompostChance() float64 {
	return 0.85
}

func (n NetherWartBlock) EncodeItem() (name string, meta int16) {
	if n.Warped {
		return "minecraft:warped_wart_block", 0
	}
	return "minecraft:nether_wart_block", 0
}

func (n NetherWartBlock) EncodeBlock() (name string, properties map[string]interface{}) {
	if n.Warped {
		return "minecraft:warped_wart_block", nil
	}
	return "minecraft:nether_wart_block", nil
}
