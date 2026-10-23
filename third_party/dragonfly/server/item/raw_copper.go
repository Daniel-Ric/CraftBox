package item

type RawCopper struct{}

func (RawCopper) SmeltInfo() SmeltInfo {
	return newOreSmeltInfo(NewStack(CopperIngot{}, 1), 0.7)
}

func (RawCopper) EncodeItem() (name string, meta int16) {
	return "minecraft:raw_copper", 0
}
