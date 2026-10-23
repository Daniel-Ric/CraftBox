package world

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
)

const defaultExplosionSize = 4

type ExplosionSource interface {
	Position() mgl64.Vec3

	Size() float64
}

type EntityExplosionSource struct {
	Entity Entity

	ExplosionSize float64
}

func (e EntityExplosionSource) Position() mgl64.Vec3 {
	return e.Entity.Position()
}

func (e EntityExplosionSource) Size() float64 {
	if e.ExplosionSize == 0 {
		return defaultExplosionSize
	}
	return e.ExplosionSize
}

type BlockExplosionSource struct {
	Block Block

	Pos cube.Pos

	ExplosionSize float64
}

func (b BlockExplosionSource) Position() mgl64.Vec3 {
	return b.Pos.Vec3Centre()
}

func (b BlockExplosionSource) Size() float64 {
	if b.ExplosionSize == 0 {
		return defaultExplosionSize
	}
	return b.ExplosionSize
}
