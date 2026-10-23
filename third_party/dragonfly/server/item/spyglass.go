package item

type Spyglass struct {
	nopReleasable
}

func (Spyglass) MaxCount() int {
	return 1
}

func (Spyglass) EncodeItem() (name string, meta int16) {
	return "minecraft:spyglass", 0
}
