package effect

import (
	"image/color"
)

var SlowFalling slowFalling

type slowFalling struct {
	nopLasting
}

func (slowFalling) RGBA() color.RGBA {
	return color.RGBA{R: 0xf3, G: 0xcf, B: 0xb9, A: 0xff}
}
