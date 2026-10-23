package effect

import (
	"image/color"
)

var FireResistance fireResistance

type fireResistance struct {
	nopLasting
}

func (fireResistance) RGBA() color.RGBA {
	return color.RGBA{R: 0xff, G: 0x99, B: 0x00, A: 0xff}
}
