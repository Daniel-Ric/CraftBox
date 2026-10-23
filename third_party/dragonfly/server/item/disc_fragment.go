package item

type DiscFragment struct{}

func (DiscFragment) EncodeItem() (name string, meta int16) {
	return "minecraft:disc_fragment_5", 0
}
