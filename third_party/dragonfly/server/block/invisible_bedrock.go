package block

type InvisibleBedrock struct {
	transparent
	solid
}

func (InvisibleBedrock) EncodeItem() (name string, meta int16) {
	return "minecraft:invisible_bedrock", 0
}

func (InvisibleBedrock) EncodeBlock() (string, map[string]any) {
	return "minecraft:invisible_bedrock", nil
}
