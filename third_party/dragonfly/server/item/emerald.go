package item

import "github.com/sandertv/gophertunnel/minecraft/text"

type Emerald struct{}

func (Emerald) EncodeItem() (name string, meta int16) {
	return "minecraft:emerald", 0
}

func (Emerald) TrimMaterial() string {
	return "emerald"
}

func (Emerald) MaterialColour() string {
	return text.Emerald
}

func (Emerald) PayableForBeacon() bool {
	return true
}
