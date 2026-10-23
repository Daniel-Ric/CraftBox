package enchantment

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

var Respiration respiration

type respiration struct{}

func (respiration) Name() string {
	return "Respiration"
}

func (respiration) MaxLevel() int {
	return 3
}

func (respiration) Cost(level int) (int, int) {
	minCost := 10 * level
	return minCost, minCost + 30
}

func (respiration) Rarity() item.EnchantmentRarity {
	return item.EnchantmentRarityRare
}

func (respiration) Chance(level int) float64 {
	return float64(level) / float64(level+1)
}

func (respiration) CompatibleWithEnchantment(item.EnchantmentType) bool {
	return true
}

func (respiration) CompatibleWithItem(i world.Item) bool {
	h, ok := i.(item.HelmetType)
	return ok && h.Helmet()
}
