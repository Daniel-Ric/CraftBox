package item

type Book struct{}

func (b Book) EnchantmentValue() int {
	return 1
}

func (Book) EncodeItem() (name string, meta int16) {
	return "minecraft:book", 0
}
