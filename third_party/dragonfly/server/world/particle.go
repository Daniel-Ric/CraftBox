package world

import (
	"github.com/go-gl/mathgl/mgl64"
)

type Particle interface {
	Spawn(w *World, pos mgl64.Vec3)
}
