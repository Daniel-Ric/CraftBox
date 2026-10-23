package item

import (
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/world"
	"time"
)

type Pufferfish struct {
	defaultFood
}

func (p Pufferfish) Consume(_ *world.Tx, c Consumer) Stack {
	c.Saturate(1, 0.2)
	c.AddEffect(effect.New(effect.Hunger, 3, 15*time.Second))
	c.AddEffect(effect.New(effect.Poison, 2, time.Minute))
	c.AddEffect(effect.New(effect.Nausea, 2, 15*time.Second))
	return Stack{}
}

func (p Pufferfish) EncodeItem() (name string, meta int16) {
	return "minecraft:pufferfish", 0
}
