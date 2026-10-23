package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Fence struct {
	Wood bool
}

const (
	fenceHeight = 1.5
	fenceInset  = 0.375
)

func (f Fence) BBox(pos cube.Pos, s world.BlockSource) []cube.BBox {
	boxes := make([]cube.BBox, 0, 2)

	connectWest, connectEast := f.Connects(pos, cube.FaceWest, s), f.Connects(pos, cube.FaceEast, s)
	connectNorth, connectSouth := f.Connects(pos, cube.FaceNorth, s), f.Connects(pos, cube.FaceSouth, s)

	if connectWest || connectEast {
		sideBox := cube.Box(0, 0, 0, 1, fenceHeight, 1).Stretch(cube.Z, -fenceInset)
		if connectWest {
			boxes = append(boxes, sideBox.ExtendTowards(cube.FaceEast, -fenceInset))
		}
		if connectEast {
			boxes = append(boxes, sideBox.ExtendTowards(cube.FaceWest, -fenceInset))
		}
	}

	if connectNorth || connectSouth {
		sideBox := cube.Box(0, 0, 0, 1, fenceHeight, 1).Stretch(cube.X, -fenceInset)
		if connectNorth {
			boxes = append(boxes, sideBox.ExtendTowards(cube.FaceSouth, -fenceInset))
		}
		if connectSouth {
			boxes = append(boxes, sideBox.ExtendTowards(cube.FaceNorth, -fenceInset))
		}
	}

	if len(boxes) == 0 {
		boxes = append(boxes, cube.Box(fenceInset, 0, fenceInset, 1-fenceInset, fenceHeight, 1-fenceInset))
	}

	return boxes
}

func (f Fence) FaceSolid(_ cube.Pos, face cube.Face, _ world.BlockSource) bool {
	return face == cube.FaceDown || face == cube.FaceUp
}

func (f Fence) Connects(pos cube.Pos, face cube.Face, src world.BlockSource) bool {
	sidePos := pos.Side(face)
	sideBlock := src.Block(sidePos)
	if fence, ok := sideBlock.Model().(Fence); ok && fence.Wood == f.Wood {
		return true
	}
	if sideBlock.Model().FaceSolid(sidePos, face.Opposite(), src) {
		return true
	}
	_, ok := sideBlock.Model().(FenceGate)
	return ok
}
