package block

type Bedrock struct {
	solid
	bassDrum

	InfiniteBurning bool
}

func (Bedrock) EncodeItem() (name string, meta int16) {
	return "minecraft:bedrock", 0
}

func (b Bedrock) EncodeBlock() (name string, properties map[string]any) {

	return "minecraft:bedrock", map[string]any{"infiniburn_bit": b.InfiniteBurning}
}

func (Bedrock) SupportsEndCrystal() bool {
	return true
}
