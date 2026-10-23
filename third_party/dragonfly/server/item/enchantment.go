package item

import (
	"github.com/df-mc/dragonfly/server/world"
	"maps"
	"slices"
	"sort"
)

type Enchantment struct {
	t   EnchantmentType
	lvl int
}

func NewEnchantment(t EnchantmentType, lvl int) Enchantment {
	if lvl < 1 {
		panic("enchantment level must never be below 1")
	}
	return Enchantment{t: t, lvl: lvl}
}

func (e Enchantment) Level() int {
	return e.lvl
}

func (e Enchantment) Type() EnchantmentType {
	return e.t
}

type EnchantmentType interface {
	Name() string

	MaxLevel() int

	Cost(level int) (int, int)

	Rarity() EnchantmentRarity

	CompatibleWithEnchantment(t EnchantmentType) bool

	CompatibleWithItem(i world.Item) bool
}

type Enchantable interface {
	EnchantmentValue() int
}

func RegisterEnchantment(id int, enchantment EnchantmentType) {
	enchantmentsMap[id] = enchantment
	enchantmentIDs[enchantment] = id
}

var (
	enchantmentsMap = map[int]EnchantmentType{}
	enchantmentIDs  = map[EnchantmentType]int{}
)

func EnchantmentByID(id int) (EnchantmentType, bool) {
	e, ok := enchantmentsMap[id]
	return e, ok
}

func EnchantmentID(e EnchantmentType) (int, bool) {
	id, ok := enchantmentIDs[e]
	return id, ok
}

func Enchantments() []EnchantmentType {
	e := slices.Collect(maps.Values(enchantmentsMap))
	sort.Slice(e, func(i, j int) bool {
		id1, _ := EnchantmentID(e[i])
		id2, _ := EnchantmentID(e[j])
		return id1 < id2
	})
	return e
}
