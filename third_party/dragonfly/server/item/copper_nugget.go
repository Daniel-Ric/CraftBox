package item

type CopperNugget struct{}

func (CopperNugget) EncodeItem() (name string, meta int16) {
	return "minecraft:copper_nugget", 0
}
