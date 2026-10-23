package effect

import (
	"image/color"
)

var Levitation levitation

type levitation struct {
	nopLasting
}

func (levitation) RGBA() color.RGBA {
	return color.RGBA{R: 0xce, G: 0xff, B: 0xff, A: 0xff}
}
