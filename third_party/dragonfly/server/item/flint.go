package item

type Flint struct{}

func (Flint) EncodeItem() (name string, meta int16) {
	return "minecraft:flint", 0
}
