package effect

import (
	"image/color"
)

var NightVision nightVision

type nightVision struct {
	nopLasting
}

func (nightVision) RGBA() color.RGBA {
	return color.RGBA{R: 0xc2, G: 0xff, B: 0x66, A: 0xff}
}
