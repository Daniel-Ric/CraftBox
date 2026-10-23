package effect

import (
	"image/color"
)

var JumpBoost jumpBoost

type jumpBoost struct {
	nopLasting
}

func (jumpBoost) RGBA() color.RGBA {
	return color.RGBA{R: 0xfd, G: 0xff, B: 0x84, A: 0xff}
}
