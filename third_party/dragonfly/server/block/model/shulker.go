package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Shulker struct {
	Facing cube.Face

	Progress int32
}

func (s Shulker) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	peak := ShulkerPhysicalPeak(s.Progress)
	return []cube.BBox{full.ExtendTowards(s.Facing, peak)}
}

func ShulkerPhysicalPeak(progress int32) float64 {
	t := float64(progress) / 10.0
	return (1.0 - (1.0-t)*(1.0-t)*(1.0-t)) * 0.5
}

func (Shulker) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
