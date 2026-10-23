package item

type GoldNugget struct{}

func (GoldNugget) EncodeItem() (name string, meta int16) {
	return "minecraft:gold_nugget", 0
}
