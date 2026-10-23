package item

type IronNugget struct{}

func (IronNugget) EncodeItem() (name string, meta int16) {
	return "minecraft:iron_nugget", 0
}
