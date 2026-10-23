package item

import "github.com/df-mc/dragonfly/server/item/potion"

type Arrow struct {
	Tip potion.Potion
}

func (a Arrow) EncodeItem() (name string, meta int16) {
	if tip := a.Tip.Uint8(); tip > 4 {
		return "minecraft:arrow", int16(tip + 1)
	}
	return "minecraft:arrow", 0
}

func (Arrow) OffHand() bool {
	return true
}
