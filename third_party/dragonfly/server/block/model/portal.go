package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Portal struct {
	Axis cube.Axis
}

func (Portal) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return nil
}

func (Portal) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
