package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type Stair struct {
	Facing cube.Direction

	UpsideDown bool

	Corner bool

	Inner bool

	Left bool
}

func (s Stair) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	b := []cube.BBox{cube.Box(0, 0, 0, 1, 0.5, 1)}
	if s.UpsideDown {
		b[0] = cube.Box(0, 0.5, 0, 1, 1, 1)
	}
	step := cube.Box(0.5, 0.5, 0.5, 0.5, 1, 0.5)
	if !s.Corner || s.Inner {
		b = append(b, step.ExtendTowards(s.Facing.Face(), 0.5).Stretch(s.Facing.RotateRight().Face().Axis(), 0.5))
	}
	if s.Corner {

		face := s.Facing.Face()
		if s.Inner {
			face = s.Facing.Opposite().Face()
		}
		b = append(b, step.ExtendTowards(face, 0.5).ExtendTowards(s.cornerFace(), 0.5))
	}
	if s.UpsideDown {
		for i := range b[1:] {
			b[i+1] = b[i+1].Translate(mgl64.Vec3{0, -0.5})
		}
	}
	return b
}

func (s Stair) FaceSolid(_ cube.Pos, face cube.Face, _ world.BlockSource) bool {

	if (face == cube.FaceUp && s.UpsideDown) || (face == cube.FaceDown && !s.UpsideDown) {
		return true
	}
	if !s.Corner {

		return face == s.Facing.Face()
	}
	if !s.Inner {

		return false
	}
	return face == s.cornerFace() || face == s.Facing.Face()
}

func (s Stair) cornerFace() cube.Face {
	if s.Left {
		return s.Facing.RotateLeft().Face()
	}
	return s.Facing.RotateRight().Face()
}
