package effect

import (
	"github.com/df-mc/dragonfly/server/world"
	"image/color"
)

var FatalPoison fatalPoison

type fatalPoison struct {
	nopLasting
}

func (fatalPoison) Apply(e world.Entity, eff Effect) {
	interval := max(25>>(eff.Level()-1), 1)
	if eff.Tick()%interval == 0 {
		if l, ok := e.(living); ok {
			l.Hurt(1, PoisonDamageSource{Fatal: true})
		}
	}
}

func (fatalPoison) RGBA() color.RGBA {
	return color.RGBA{R: 0x4e, G: 0x93, B: 0x31, A: 0xff}
}
