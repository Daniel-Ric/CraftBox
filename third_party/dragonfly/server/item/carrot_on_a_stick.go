package item

type CarrotOnAStick struct{}

func (CarrotOnAStick) MaxCount() int {
	return 1
}

func (CarrotOnAStick) EncodeItem() (name string, meta int16) {
	return "minecraft:carrot_on_a_stick", 0
}
