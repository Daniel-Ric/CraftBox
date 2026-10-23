package effect

import (
	"image/color"
	"math"
)

var MiningFatigue miningFatigue

type miningFatigue struct {
	nopLasting
}

func (miningFatigue) Multiplier(lvl int) float64 {
	return math.Pow(3, float64(lvl))
}

func (miningFatigue) RGBA() color.RGBA {
	return color.RGBA{R: 0x4a, G: 0x42, B: 0x17, A: 0xff}
}
