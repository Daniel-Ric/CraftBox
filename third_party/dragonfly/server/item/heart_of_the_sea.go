package item

type HeartOfTheSea struct{}

func (HeartOfTheSea) EncodeItem() (name string, meta int16) {
	return "minecraft:heart_of_the_sea", 0
}
