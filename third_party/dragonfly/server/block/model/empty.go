package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Empty struct{}

func (Empty) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return nil
}

func (Empty) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
