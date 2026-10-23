package item

import "github.com/sandertv/gophertunnel/minecraft/text"

type AmethystShard struct{}

func (AmethystShard) EncodeItem() (name string, meta int16) {
	return "minecraft:amethyst_shard", 0
}

func (AmethystShard) TrimMaterial() string {
	return "amethyst"
}

func (AmethystShard) MaterialColour() string {
	return text.Amethyst
}
