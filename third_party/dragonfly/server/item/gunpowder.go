package item

type Gunpowder struct{}

func (Gunpowder) EncodeItem() (name string, meta int16) {
	return "minecraft:gunpowder", 0
}
