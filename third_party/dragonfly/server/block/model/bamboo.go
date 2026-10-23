package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Bamboo struct {
	Thick bool
}

func (b Bamboo) BBox(pos cube.Pos, s world.BlockSource) []cube.BBox {

	size := 0.5 + 2.0/16.0
	if b.Thick {
		size = 0.5 + 3.0/16.0
	}
	offset := randomOffset(pos, -0.25, 0.25, 16)
	return []cube.BBox{cube.Box(0.5, 0, 0.5, size, 1, size).Translate(offset)}
}

func (b Bamboo) FaceSolid(pos cube.Pos, face cube.Face, s world.BlockSource) bool {
	return false
}
