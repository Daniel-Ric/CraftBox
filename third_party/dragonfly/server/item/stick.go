package item

import "time"

type Stick struct{}

func (Stick) FuelInfo() FuelInfo {
	return newFuelInfo(time.Second * 5)
}

func (s Stick) EncodeItem() (name string, meta int16) {
	return "minecraft:stick", 0
}
