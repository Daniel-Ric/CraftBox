package item

type BlazePowder struct{}

func (BlazePowder) EncodeItem() (name string, meta int16) {
	return "minecraft:blaze_powder", 0
}
