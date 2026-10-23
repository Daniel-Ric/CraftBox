package item

type Clock struct{}

func (w Clock) EncodeItem() (name string, meta int16) {
	return "minecraft:clock", 0
}
