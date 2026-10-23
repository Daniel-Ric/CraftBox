package item

type GlisteringMelonSlice struct{}

func (GlisteringMelonSlice) EncodeItem() (name string, meta int16) {
	return "minecraft:glistering_melon_slice", 0
}
