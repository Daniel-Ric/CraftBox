package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Door struct {
	Facing cube.Direction

	Open bool

	Right bool
}

func (d Door) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	if d.Open {
		if d.Right {
			return []cube.BBox{full.ExtendTowards(d.Facing.RotateLeft().Face(), -0.8125)}
		}
		return []cube.BBox{full.ExtendTowards(d.Facing.RotateRight().Face(), -0.8125)}
	}
	return []cube.BBox{full.ExtendTowards(d.Facing.Face(), -0.8125)}
}

func (d Door) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
