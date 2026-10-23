package effect

import (
	"image/color"
)

var Nausea nausea

type nausea struct {
	nopLasting
}

func (nausea) RGBA() color.RGBA {
	return color.RGBA{R: 0x55, G: 0x1d, B: 0x4a, A: 0xff}
}
