package item

type PrismarineShard struct{}

func (PrismarineShard) EncodeItem() (name string, meta int16) {
	return "minecraft:prismarine_shard", 0
}
