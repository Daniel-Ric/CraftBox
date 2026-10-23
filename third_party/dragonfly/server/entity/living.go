package entity

import (
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type Living interface {
	world.Entity

	Health() float64

	MaxHealth() float64

	SetMaxHealth(v float64)

	Dead() bool

	Hurt(damage float64, src world.DamageSource) (n float64, vulnerable bool)

	Heal(health float64, src world.HealingSource) float64

	KnockBack(src mgl64.Vec3, force, height float64)

	Velocity() mgl64.Vec3

	SetVelocity(velocity mgl64.Vec3)

	AddEffect(e effect.Effect)

	RemoveEffect(e effect.Type)

	Effects() []effect.Effect

	Speed() float64

	SetSpeed(float64)
}
