package effect

import (
	"github.com/df-mc/dragonfly/server/world"
	"image/color"
)

var Slowness slowness

type slowness struct {
	nopLasting
}

func (slowness) Start(e world.Entity, lvl int) {
	slowness := 1 - float64(lvl)*0.15
	if slowness <= 0 {
		slowness = 0.00001
	}
	if l, ok := e.(living); ok {
		l.SetSpeed(l.Speed() * slowness)
	}
}

func (slowness) End(e world.Entity, lvl int) {
	slowness := 1 - float64(lvl)*0.15
	if slowness <= 0 {
		slowness = 0.00001
	}
	if l, ok := e.(living); ok {
		l.SetSpeed(l.Speed() / slowness)
	}
}

func (slowness) RGBA() color.RGBA {
	return color.RGBA{R: 0x8b, G: 0xaf, B: 0xe0, A: 0xff}
}
