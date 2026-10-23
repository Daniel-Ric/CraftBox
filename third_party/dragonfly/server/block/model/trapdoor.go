package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Trapdoor struct {
	Facing cube.Direction

	Open, Top bool
}

func (t Trapdoor) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	if t.Open {
		return []cube.BBox{full.ExtendTowards(t.Facing.Face(), -0.8125)}
	} else if t.Top {
		return []cube.BBox{cube.Box(0, 0.8125, 0, 1, 1, 1)}
	}
	return []cube.BBox{cube.Box(0, 0, 0, 1, 0.1875, 1)}
}

func (t Trapdoor) FaceSolid(_ cube.Pos, face cube.Face, _ world.BlockSource) bool {
	if t.Open {
		return t.Facing.Face().Opposite() == face
	} else if t.Top {
		return face == cube.FaceUp
	}
	return face == cube.FaceDown
}
