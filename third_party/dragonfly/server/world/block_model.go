package world

import (
	"github.com/df-mc/dragonfly/server/block/cube"
)

type BlockModel interface {
	BBox(pos cube.Pos, s BlockSource) []cube.BBox

	FaceSolid(pos cube.Pos, face cube.Face, s BlockSource) bool
}

type unknownModel struct{}

func (u unknownModel) BBox(cube.Pos, BlockSource) []cube.BBox {
	return []cube.BBox{cube.Box(0, 0, 0, 1, 1, 1)}
}

func (u unknownModel) FaceSolid(cube.Pos, cube.Face, BlockSource) bool {
	return true
}
