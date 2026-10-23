package item

type Slimeball struct{}

func (Slimeball) EncodeItem() (name string, meta int16) {
	return "minecraft:slime_ball", 0
}
