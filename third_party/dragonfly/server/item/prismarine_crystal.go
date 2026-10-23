package item

type PrismarineCrystals struct{}

func (p PrismarineCrystals) EncodeItem() (name string, meta int16) {
	return "minecraft:prismarine_crystals", 0
}
