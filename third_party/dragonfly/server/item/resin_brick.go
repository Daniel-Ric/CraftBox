package item

import "github.com/sandertv/gophertunnel/minecraft/text"

type ResinBrick struct{}

func (ResinBrick) EncodeItem() (name string, meta int16) {
	return "minecraft:resin_brick", 0
}

func (ResinBrick) TrimMaterial() string {
	return "resin"
}

func (ResinBrick) MaterialColour() string {
	return text.Resin
}
