package item

import (
	"time"

	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/world"
)

type HoneyBottle struct{}

func (HoneyBottle) MaxCount() int {
	return 16
}

func (HoneyBottle) AlwaysConsumable() bool {
	return true
}

func (HoneyBottle) ConsumeDuration() time.Duration {
	return DefaultConsumeDuration * 5 / 4
}

func (HoneyBottle) Consume(_ *world.Tx, c Consumer) Stack {
	c.Saturate(6, 1.2)
	c.RemoveEffect(effect.Poison)
	return NewStack(GlassBottle{}, 1)
}

func (HoneyBottle) EncodeItem() (name string, meta int16) {
	return "minecraft:honey_bottle", 0
}
