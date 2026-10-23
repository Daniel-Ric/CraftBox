package effect

import (
	"github.com/df-mc/dragonfly/server/world"
	"image/color"
)

var InstantHealth instantHealth

type instantHealth struct{}

func (i instantHealth) Apply(e world.Entity, eff Effect) {
	base := 2 << eff.Level()
	if l, ok := e.(living); ok {
		l.Heal(float64(base)*eff.potency, InstantHealingSource{})
	}
}

func (instantHealth) RGBA() color.RGBA {
	return color.RGBA{R: 0xf8, G: 0x24, B: 0x23, A: 0xff}
}

type InstantHealingSource struct{}

func (InstantHealingSource) HealingSource() {}
