package item

type EnchantmentRarity interface {
	Name() string

	Cost() int

	Weight() int
}

var (
	EnchantmentRarityCommon enchantmentRarityCommon

	EnchantmentRarityUncommon enchantmentRarityUncommon

	EnchantmentRarityRare enchantmentRarityRare

	EnchantmentRarityVeryRare enchantmentRarityVeryRare
)

type enchantmentRarityCommon struct{}

func (enchantmentRarityCommon) Name() string { return "Common" }
func (enchantmentRarityCommon) Cost() int    { return 1 }
func (enchantmentRarityCommon) Weight() int  { return 10 }

type enchantmentRarityUncommon struct{}

func (enchantmentRarityUncommon) Name() string { return "Uncommon" }
func (enchantmentRarityUncommon) Cost() int    { return 2 }
func (enchantmentRarityUncommon) Weight() int  { return 5 }

type enchantmentRarityRare struct{}

func (enchantmentRarityRare) Name() string { return "Rare" }
func (enchantmentRarityRare) Cost() int    { return 4 }
func (enchantmentRarityRare) Weight() int  { return 2 }

type enchantmentRarityVeryRare struct{}

func (enchantmentRarityVeryRare) Name() string { return "Very Rare" }
func (enchantmentRarityVeryRare) Cost() int    { return 8 }
func (enchantmentRarityVeryRare) Weight() int  { return 1 }
