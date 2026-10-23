package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Lantern struct {
	Hanging bool
}

func (l Lantern) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	if l.Hanging {
		return []cube.BBox{cube.Box(0.3125, 0.125, 0.3125, 0.6875, 0.625, 0.6875)}
	}
	return []cube.BBox{cube.Box(0.3125, 0, 0.3125, 0.6875, 0.5, 0.6875)}
}

func (l Lantern) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
