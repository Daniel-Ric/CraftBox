package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type TilledGrass struct{}

func (TilledGrass) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{full.ExtendTowards(cube.FaceUp, -0.0625)}
}

func (TilledGrass) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return true
}
