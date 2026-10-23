package enchantment

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

var Multishot multishot

type multishot struct{}

func (multishot) Name() string {
	return "Multishot"
}

func (multishot) MaxLevel() int {
	return 1
}

func (m multishot) Cost(level int) (int, int) {
	return 20, 50
}

func (multishot) Rarity() item.EnchantmentRarity {
	return item.EnchantmentRarityRare
}

func (multishot) CompatibleWithEnchantment(t item.EnchantmentType) bool {
	return t != Piercing
}

func (multishot) CompatibleWithItem(i world.Item) bool {
	_, ok := i.(item.Crossbow)
	return ok
}

func (multishot) MultipleProjectiles() bool {
	return true
}
