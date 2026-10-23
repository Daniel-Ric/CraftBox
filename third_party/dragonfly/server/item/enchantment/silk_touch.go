package enchantment

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

var SilkTouch silkTouch

type silkTouch struct{}

func (silkTouch) Name() string {
	return "Silk Touch"
}

func (silkTouch) MaxLevel() int {
	return 1
}

func (silkTouch) Cost(int) (int, int) {
	return 15, 65
}

func (silkTouch) Rarity() item.EnchantmentRarity {
	return item.EnchantmentRarityVeryRare
}

func (silkTouch) CompatibleWithEnchantment(t item.EnchantmentType) bool {
	return t != Fortune
}

func (silkTouch) CompatibleWithItem(i world.Item) bool {
	t, ok := i.(item.Tool)
	return ok && (t.ToolType() != item.TypeSword && t.ToolType() != item.TypeNone)
}
