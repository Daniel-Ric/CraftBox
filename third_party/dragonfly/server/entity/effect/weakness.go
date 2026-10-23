package effect

import (
	"image/color"
)

var Weakness weakness

type weakness struct {
	nopLasting
}

func (weakness) Multiplier(lvl int) float64 {
	v := 0.2 * float64(lvl)
	if v > 1 {
		v = 1
	}
	return v
}

func (weakness) RGBA() color.RGBA {
	return color.RGBA{R: 0x48, G: 0x4d, B: 0x48, A: 0xff}
}
