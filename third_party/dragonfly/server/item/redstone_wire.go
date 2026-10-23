package item

import "github.com/sandertv/gophertunnel/minecraft/text"

type RedstoneWire struct{}

func (RedstoneWire) EncodeItem() (name string, meta int16) {
	return "minecraft:redstone", 0
}

func (RedstoneWire) TrimMaterial() string {
	return "redstone"
}

func (RedstoneWire) MaterialColour() string {
	return text.Redstone
}
