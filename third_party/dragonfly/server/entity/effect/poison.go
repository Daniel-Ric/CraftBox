package effect

import (
	"github.com/df-mc/dragonfly/server/world"
	"image/color"
)

var Poison poison

type poison struct {
	nopLasting
}

func (poison) Apply(e world.Entity, eff Effect) {
	interval := max(25>>(eff.Level()-1), 1)
	if eff.Tick()%interval == 0 {
		if l, ok := e.(living); ok && l.Health() > 1 {
			l.Hurt(1, PoisonDamageSource{})
		}
	}
}

func (poison) RGBA() color.RGBA {
	return color.RGBA{R: 0x87, G: 0xa3, B: 0x63, A: 0xff}
}

type PoisonDamageSource struct {
	Fatal bool
}

func (PoisonDamageSource) ReducedByResistance() bool { return true }
func (PoisonDamageSource) ReducedByArmour() bool     { return false }
func (PoisonDamageSource) Fire() bool                { return false }
func (PoisonDamageSource) IgnoreTotem() bool         { return false }
