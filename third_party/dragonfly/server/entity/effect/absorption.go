package effect

import (
	"github.com/df-mc/dragonfly/server/world"
	"image/color"
)

var Absorption absorption

type absorption struct {
	nopLasting
}

func (absorption) Start(e world.Entity, lvl int) {
	if i, ok := e.(interface {
		SetAbsorption(health float64)
	}); ok {
		i.SetAbsorption(4 * float64(lvl))
	}
}

func (absorption) End(e world.Entity, _ int) {
	if i, ok := e.(interface {
		SetAbsorption(health float64)
	}); ok {
		i.SetAbsorption(0)
	}
}

func (absorption) RGBA() color.RGBA {
	return color.RGBA{R: 0x25, G: 0x52, B: 0xa5, A: 0xff}
}
