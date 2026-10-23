package effect

import (
	"github.com/df-mc/dragonfly/server/world"
	"image/color"
)

var Regeneration regeneration

type regeneration struct {
	nopLasting
}

func (regeneration) Apply(e world.Entity, eff Effect) {
	interval := max(50>>(eff.Level()-1), 1)
	if eff.Tick()%interval == 0 {
		if l, ok := e.(living); ok {
			l.Heal(1, RegenerationHealingSource{})
		}
	}
}

func (regeneration) RGBA() color.RGBA {
	return color.RGBA{R: 0xcd, G: 0x5c, B: 0xab, A: 0xff}
}

type RegenerationHealingSource struct{}

func (RegenerationHealingSource) HealingSource() {}
