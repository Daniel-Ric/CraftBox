package item

type GlowstoneDust struct{}

func (g GlowstoneDust) EncodeItem() (name string, meta int16) {
	return "minecraft:glowstone_dust", 0
}
