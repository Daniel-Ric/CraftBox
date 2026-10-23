package item

type NetheriteScrap struct{}

func (NetheriteScrap) EncodeItem() (name string, meta int16) {
	return "minecraft:netherite_scrap", 0
}
