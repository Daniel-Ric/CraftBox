package block

type ReinforcedDeepslate struct {
	solid
	bassDrum
}

func (r ReinforcedDeepslate) BreakInfo() BreakInfo {
	return newBreakInfo(55, alwaysHarvestable, nothingEffective, oneOf(r)).withBlastResistance(1200)
}

func (ReinforcedDeepslate) EncodeItem() (name string, meta int16) {
	return "minecraft:reinforced_deepslate", 0
}

func (ReinforcedDeepslate) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:reinforced_deepslate", nil
}
