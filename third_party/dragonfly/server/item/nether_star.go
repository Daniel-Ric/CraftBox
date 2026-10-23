package item

type NetherStar struct{}

func (NetherStar) EncodeItem() (name string, meta int16) {
	return "minecraft:nether_star", 0
}

func (NetherStar) BlastProof() bool {
	return true
}
