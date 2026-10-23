package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type DecoratedPot struct{}

func (DecoratedPot) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{cube.Box(0.025, 0, 0.025, 0.975, 1, 0.975)}
}

func (DecoratedPot) FaceSolid(_ cube.Pos, face cube.Face, _ world.BlockSource) bool {
	return face.Axis() == cube.Y
}
