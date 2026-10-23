package skin

import (
	"image"
	"image/color"
)

const (
	AnimationHead AnimationType = iota

	AnimationBody32x32

	AnimationBody128x128
)

type AnimationType int

type Animation struct {
	w, h  int
	aType AnimationType

	Pix []uint8

	FrameCount int

	AnimationExpression int
}

func NewAnimation(width, height int, expression int, animationType AnimationType) Animation {
	return Animation{
		w:                   width,
		h:                   height,
		aType:               animationType,
		Pix:                 make([]uint8, width*height*4),
		FrameCount:          1,
		AnimationExpression: expression,
	}
}

func (a Animation) Type() AnimationType {
	return a.aType
}

func (a Animation) ColorModel() color.Model {
	return color.RGBAModel
}

func (a Animation) Bounds() image.Rectangle {
	return image.Rectangle{
		Max: image.Point{X: a.w, Y: a.h},
	}
}

func (a Animation) At(x, y int) color.Color {
	if x < 0 || y < 0 || x >= a.w || y >= a.h {
		panic("pixel coordinates out of bounds")
	}
	offset := x*4 + a.w*y*4
	return color.RGBA{
		R: a.Pix[offset],
		G: a.Pix[offset+1],
		B: a.Pix[offset+2],
		A: a.Pix[offset+3],
	}
}
