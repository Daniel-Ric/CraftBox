package item

type RabbitFoot struct{}

func (RabbitFoot) EncodeItem() (name string, meta int16) {
	return "minecraft:rabbit_foot", 0
}
