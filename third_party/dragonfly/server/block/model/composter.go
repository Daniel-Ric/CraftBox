package model

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Composter struct {
	Level int
}

func (c Composter) BBox(_ cube.Pos, _ world.BlockSource) []cube.BBox {
	compostHeight := math.Abs(math.Min(float64(c.Level), 7)*0.125 - 0.0625)
	return []cube.BBox{
		cube.Box(0, 0, 0, 1, 1, 0.125),
		cube.Box(0, 0, 0.875, 1, 1, 1),
		cube.Box(0.875, 0, 0, 1, 1, 1),
		cube.Box(0, 0, 0, 0.125, 1, 1),
		cube.Box(0.125, 0, 0.125, 0.875, 0.125+compostHeight, 0.875),
	}
}

func (Composter) FaceSolid(_ cube.Pos, face cube.Face, _ world.BlockSource) bool {
	return face != cube.FaceUp
}
