package protocol

type EnchantmentOption struct {
	Cost uint8

	Enchantments ItemEnchantments

	Name string

	RecipeNetworkID uint32
}

func (x *EnchantmentOption) Marshal(r IO) {
	r.Uint8(&x.Cost)
	Single(r, &x.Enchantments)
	r.String(&x.Name)
	r.Varuint32(&x.RecipeNetworkID)
}

type ItemEnchantments struct {
	Slot int32

	Enchantments [3][]EnchantmentInstance
}

func (x *ItemEnchantments) Marshal(r IO) {
	r.Int32(&x.Slot)
	for i := 0; i < 3; i++ {
		Slice(r, &x.Enchantments[i])
	}
}

type EnchantmentInstance struct {
	Type  byte
	Level byte
}

func (x *EnchantmentInstance) Marshal(r IO) {
	r.Uint8(&x.Type)
	r.Uint8(&x.Level)
}
