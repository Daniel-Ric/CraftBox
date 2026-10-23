package item

type Totem struct{}

func (Totem) MaxCount() int {
	return 1
}

func (Totem) EncodeItem() (name string, meta int16) {
	return "minecraft:totem_of_undying", 0
}

func (Totem) OffHand() bool {
	return true
}
