package item

import "github.com/sandertv/gophertunnel/minecraft/text"

type IronIngot struct{}

func (IronIngot) EncodeItem() (name string, meta int16) {
	return "minecraft:iron_ingot", 0
}

func (IronIngot) TrimMaterial() string {
	return "iron"
}

func (IronIngot) MaterialColour() string {
	return text.Iron
}

func (IronIngot) PayableForBeacon() bool {
	return true
}
