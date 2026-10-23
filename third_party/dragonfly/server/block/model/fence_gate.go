package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type FenceGate struct {
	Facing cube.Direction

	Open bool
}

func (f FenceGate) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	if f.Open {
		return nil
	}
	return []cube.BBox{cube.Box(0, 0, 0, 1, 1.5, 1).Stretch(f.Facing.Face().Axis(), -0.375)}
}

func (f FenceGate) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
