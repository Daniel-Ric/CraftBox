package block

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
)

type Wool struct {
	solid

	Colour item.Colour
}

func (w Wool) Instrument() sound.Instrument {
	return sound.Guitar()
}

func (w Wool) FlammabilityInfo() FlammabilityInfo {
	return newFlammabilityInfo(30, 60, true)
}

func (w Wool) BreakInfo() BreakInfo {
	return newBreakInfo(0.8, alwaysHarvestable, shearsEffective, oneOf(w))
}

func (w Wool) EncodeItem() (name string, meta int16) {
	return "minecraft:" + w.Colour.String() + "_wool", 0
}

func (w Wool) EncodeBlock() (name string, properties map[string]any) {
	return "minecraft:" + w.Colour.String() + "_wool", nil
}

func allWool() []world.Block {
	b := make([]world.Block, 0, 16)
	for _, c := range item.Colours() {
		b = append(b, Wool{Colour: c})
	}
	return b
}
