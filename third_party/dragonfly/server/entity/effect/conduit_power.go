package effect

import (
	"image/color"
)

var ConduitPower conduitPower

type conduitPower struct {
	nopLasting
}

func (conduitPower) Multiplier(lvl int) float64 {
	v := 1 - float64(lvl)*0.1
	if v < 0 {
		v = 0
	}
	return v
}

func (conduitPower) RGBA() color.RGBA {
	return color.RGBA{R: 0x1d, G: 0xc2, B: 0xd1, A: 0xff}
}
