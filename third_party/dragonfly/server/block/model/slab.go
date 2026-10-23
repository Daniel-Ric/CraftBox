package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Slab struct {
	Double, Top bool
}

func (s Slab) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	if s.Double {
		return []cube.BBox{full}
	}
	if s.Top {
		return []cube.BBox{cube.Box(0, 0.5, 0, 1, 1, 1)}
	}
	return []cube.BBox{cube.Box(0, 0, 0, 1, 0.5, 1)}
}

func (s Slab) FaceSolid(_ cube.Pos, face cube.Face, _ world.BlockSource) bool {
	if s.Double {
		return true
	} else if s.Top {
		return face == cube.FaceUp
	}
	return face == cube.FaceDown
}
