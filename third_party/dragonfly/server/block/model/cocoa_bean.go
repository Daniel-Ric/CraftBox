package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type CocoaBean struct {
	Facing cube.Direction

	Age int
}

func (c CocoaBean) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{full.
		Stretch(c.Facing.RotateRight().Face().Axis(), -(6-float64(c.Age))/16).
		ExtendTowards(cube.FaceUp, -0.25).
		ExtendTowards(cube.FaceDown, -((7-float64(c.Age)*2)/16)).
		ExtendTowards(c.Facing.Face(), -0.0625).
		ExtendTowards(c.Facing.Opposite().Face(), -((11 - float64(c.Age)*2) / 16))}
}

func (c CocoaBean) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
