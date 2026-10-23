package effect

import (
	"image/color"
)

var Blindness blindness

type blindness struct {
	nopLasting
}

func (blindness) RGBA() color.RGBA {
	return color.RGBA{R: 0x1f, G: 0x1f, B: 0x23, A: 0xff}
}
