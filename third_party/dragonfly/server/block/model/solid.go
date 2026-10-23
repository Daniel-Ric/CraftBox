package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Solid struct{}

var full = cube.Box(0, 0, 0, 1, 1, 1)

func (Solid) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{full}
}

func (Solid) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return true
}
