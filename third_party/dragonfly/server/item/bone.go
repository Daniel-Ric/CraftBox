package item

type Bone struct{}

func (Bone) EncodeItem() (name string, meta int16) {
	return "minecraft:bone", 0
}
