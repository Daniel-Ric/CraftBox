package world

import "github.com/go-gl/mathgl/mgl64"

type Sound interface {
	Play(w *World, pos mgl64.Vec3)
}
