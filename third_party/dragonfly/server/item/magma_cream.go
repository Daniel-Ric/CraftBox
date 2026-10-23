package item

type MagmaCream struct{}

func (m MagmaCream) EncodeItem() (name string, meta int16) {
	return "minecraft:magma_cream", 0
}
