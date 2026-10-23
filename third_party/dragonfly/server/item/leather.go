package item

type Leather struct{}

func (Leather) EncodeItem() (name string, meta int16) {
	return "minecraft:leather", 0
}
