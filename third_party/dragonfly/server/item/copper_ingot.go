package item

import "github.com/sandertv/gophertunnel/minecraft/text"

type CopperIngot struct{}

func (c CopperIngot) EncodeItem() (name string, meta int16) {
	return "minecraft:copper_ingot", 0
}

func (CopperIngot) TrimMaterial() string {
	return "copper"
}

func (CopperIngot) MaterialColour() string {
	return text.Copper
}
