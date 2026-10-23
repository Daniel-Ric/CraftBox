package item

type Feather struct{}

func (Feather) EncodeItem() (name string, meta int16) {
	return "minecraft:feather", 0
}
