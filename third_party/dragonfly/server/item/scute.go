package item

type Scute struct{}

func (Scute) EncodeItem() (name string, meta int16) {
	return "minecraft:turtle_scute", 0
}
