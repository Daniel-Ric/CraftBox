package item

type Brick struct{}

func (b Brick) EncodeItem() (name string, meta int16) {
	return "minecraft:brick", 0
}

func (Brick) PotDecoration() bool {
	return true
}
