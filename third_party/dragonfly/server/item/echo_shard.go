package item

type EchoShard struct{}

func (EchoShard) EncodeItem() (name string, meta int16) {
	return "minecraft:echo_shard", 0
}
