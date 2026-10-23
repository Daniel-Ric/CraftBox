package item

import "github.com/sandertv/gophertunnel/minecraft/text"

type GoldIngot struct{}

func (GoldIngot) EncodeItem() (name string, meta int16) {
	return "minecraft:gold_ingot", 0
}

func (GoldIngot) TrimMaterial() string {
	return "gold"
}

func (GoldIngot) MaterialColour() string {
	return text.Gold
}

func (GoldIngot) PayableForBeacon() bool {
	return true
}
