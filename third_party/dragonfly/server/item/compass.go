package item

type Compass struct{}

func (Compass) EncodeItem() (name string, meta int16) {
	return "minecraft:compass", 0
}
