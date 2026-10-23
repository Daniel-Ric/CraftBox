package item

type PhantomMembrane struct{}

func (PhantomMembrane) EncodeItem() (name string, meta int16) {
	return "minecraft:phantom_membrane", 0
}
