package item

type PoppedChorusFruit struct{}

func (PoppedChorusFruit) EncodeItem() (name string, meta int16) {
	return "minecraft:popped_chorus_fruit", 0
}
