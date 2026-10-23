package item

import "time"

type Bowl struct{}

func (Bowl) FuelInfo() FuelInfo {
	return newFuelInfo(time.Second * 10)
}

func (Bowl) EncodeItem() (name string, meta int16) {
	return "minecraft:bowl", 0
}
