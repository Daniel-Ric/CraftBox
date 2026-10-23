package item

import "time"

type Charcoal struct{}

func (Charcoal) FuelInfo() FuelInfo {
	return newFuelInfo(time.Second * 80)
}

func (Charcoal) EncodeItem() (name string, meta int16) {
	return "minecraft:charcoal", 0
}
