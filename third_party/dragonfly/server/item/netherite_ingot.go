package item

import "github.com/sandertv/gophertunnel/minecraft/text"

type NetheriteIngot struct{}

func (NetheriteIngot) EncodeItem() (name string, meta int16) {
	return "minecraft:netherite_ingot", 0
}

func (NetheriteIngot) TrimMaterial() string {
	return "netherite"
}

func (NetheriteIngot) MaterialColour() string {
	return text.Netherite
}

func (NetheriteIngot) PayableForBeacon() bool {
	return true
}
