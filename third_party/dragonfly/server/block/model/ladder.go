package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Ladder struct {
	Facing cube.Direction
}

func (l Ladder) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{full.ExtendTowards(l.Facing.Face(), -0.8125)}
}

func (l Ladder) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
