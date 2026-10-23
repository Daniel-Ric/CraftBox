package effect

import (
	"image/color"
)

var Darkness darkness

type darkness struct {
	nopLasting
}

func (darkness) RGBA() color.RGBA {
	return color.RGBA{R: 0x29, G: 0x27, B: 0x21, A: 0xff}
}
