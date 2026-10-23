package entity

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type Eyed interface {
	EyeHeight() float64
}

func EyePosition(e world.Entity) mgl64.Vec3 {
	pos := e.Position()
	if eyed, ok := e.(Eyed); ok {
		pos[1] += eyed.EyeHeight()
	}
	return pos
}
