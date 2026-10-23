package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Leaves struct{}

func (Leaves) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{full}
}

func (Leaves) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
