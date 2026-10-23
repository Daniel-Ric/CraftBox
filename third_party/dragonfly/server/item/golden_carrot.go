package item

import "github.com/df-mc/dragonfly/server/world"

type GoldenCarrot struct {
	defaultFood
}

func (GoldenCarrot) Consume(_ *world.Tx, c Consumer) Stack {
	c.Saturate(6, 14.4)
	return Stack{}
}

func (GoldenCarrot) EncodeItem() (name string, meta int16) {
	return "minecraft:golden_carrot", 0
}
