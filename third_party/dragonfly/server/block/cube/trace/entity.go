package trace

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type EntityResult struct {
	bb   cube.BBox
	pos  mgl64.Vec3
	face cube.Face

	entity world.Entity
}

func (r EntityResult) BBox() cube.BBox {
	return r.bb
}

func (r EntityResult) Position() mgl64.Vec3 {
	return r.pos
}

func (r EntityResult) Face() cube.Face {
	return r.face
}

func (r EntityResult) Entity() world.Entity {
	return r.entity
}

func EntityIntercept(e world.Entity, start, end mgl64.Vec3) (result EntityResult, ok bool) {
	bb := e.H().Type().BBox(e).Translate(e.Position()).Grow(0.3)

	r, ok := BBoxIntercept(bb, start, end)
	if !ok {
		return
	}

	return EntityResult{bb: bb, pos: r.Position(), face: r.Face(), entity: e}, true
}
