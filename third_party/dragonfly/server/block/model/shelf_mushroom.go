package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type ShelfMushroom struct {
	Facing cube.Direction

	Large bool
}

func (s ShelfMushroom) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	depth := 0.5
	if s.Large {
		depth = 0.75
	}
	return []cube.BBox{full.
		ExtendTowards(cube.FaceUp, -0.75).
		ExtendTowards(s.Facing.Face(), -(1 - depth))}
}

func (ShelfMushroom) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
