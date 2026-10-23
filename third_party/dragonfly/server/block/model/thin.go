package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Thin struct{}

const (
	thinHeight = 1
	thinInset  = 7.0 / 16.0
)

func (t Thin) BBox(pos cube.Pos, s world.BlockSource) []cube.BBox {
	boxes := make([]cube.BBox, 0, 2)

	connectWest, connectEast := t.Connects(pos, cube.FaceWest, s), t.Connects(pos, cube.FaceEast, s)
	if connectWest || connectEast {
		box := cube.Box(0, 0, 0, 1, thinHeight, 1).Stretch(cube.Z, -thinInset)
		if !connectWest {
			box = box.ExtendTowards(cube.FaceWest, -thinInset)
		} else if !connectEast {
			box = box.ExtendTowards(cube.FaceEast, -thinInset)
		}
		boxes = append(boxes, box)
	}

	connectNorth, connectSouth := t.Connects(pos, cube.FaceNorth, s), t.Connects(pos, cube.FaceSouth, s)
	if connectNorth || connectSouth {
		box := cube.Box(0, 0, 0, 1, thinHeight, 1).Stretch(cube.X, -thinInset)
		if !connectNorth {
			box = box.ExtendTowards(cube.FaceNorth, -thinInset)
		} else if !connectSouth {
			box = box.ExtendTowards(cube.FaceSouth, -thinInset)
		}
		boxes = append(boxes, box)
	}

	if len(boxes) == 0 {
		boxes = append(boxes, cube.Box(0, 0, 0, 1, thinHeight, 1).Stretch(cube.X, -thinInset).Stretch(cube.Z, -thinInset))
	}

	return boxes
}

func (t Thin) FaceSolid(_ cube.Pos, face cube.Face, _ world.BlockSource) bool {
	return face == cube.FaceDown
}

func (t Thin) Connects(pos cube.Pos, face cube.Face, s world.BlockSource) bool {
	sidePos := pos.Side(face)
	sideBlock := s.Block(sidePos)
	_, isThin := sideBlock.Model().(Thin)
	_, isWall := sideBlock.Model().(Wall)
	return isThin || isWall || sideBlock.Model().FaceSolid(sidePos, face.Opposite(), s)
}
