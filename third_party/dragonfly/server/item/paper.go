package item

type Paper struct{}

func (Paper) EncodeItem() (name string, meta int16) {
	return "minecraft:paper", 0
}
