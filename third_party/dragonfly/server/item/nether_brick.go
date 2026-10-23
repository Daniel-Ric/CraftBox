package item

type NetherBrick struct{}

func (NetherBrick) EncodeItem() (name string, meta int16) {
	return "minecraft:netherbrick", 0
}
