package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Hopper struct{}

func (h Hopper) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	bbox := []cube.BBox{full.ExtendTowards(cube.FaceUp, -0.375)}
	for _, f := range cube.HorizontalFaces() {
		bbox = append(bbox, full.ExtendTowards(f, -0.875))
	}
	return bbox
}

func (Hopper) FaceSolid(_ cube.Pos, face cube.Face, _ world.BlockSource) bool {
	return face == cube.FaceUp
}
