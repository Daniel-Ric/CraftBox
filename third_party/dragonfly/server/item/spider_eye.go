package item

import (
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/world"
	"time"
)

type SpiderEye struct {
	defaultFood
}

func (SpiderEye) Consume(_ *world.Tx, c Consumer) Stack {
	c.Saturate(2, 3.2)
	c.AddEffect(effect.New(effect.Poison, 1, time.Second*5))
	return Stack{}
}

func (SpiderEye) EncodeItem() (name string, meta int16) {
	return "minecraft:spider_eye", 0
}
