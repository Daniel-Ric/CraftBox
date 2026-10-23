package item

type DragonBreath struct{}

func (DragonBreath) EncodeItem() (name string, meta int16) {
	return "minecraft:dragon_breath", 0
}
