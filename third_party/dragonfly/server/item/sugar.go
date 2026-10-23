package item

type Sugar struct{}

func (Sugar) EncodeItem() (name string, meta int16) {
	return "minecraft:sugar", 0
}
