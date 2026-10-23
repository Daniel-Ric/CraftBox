package item

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type GlassBottle struct{}

type bottleFiller interface {
	FillBottle() (world.Block, Stack, bool)
}

func (g GlassBottle) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, _ User, ctx *UseContext) bool {
	bl := tx.Block(pos)
	if b, ok := bl.(bottleFiller); ok {
		var res world.Block
		if res, ctx.NewItem, ok = b.FillBottle(); ok {
			ctx.SubtractFromCount(1)
			if res != bl {

				tx.SetBlock(pos, res, nil)
			}
			return true
		}
	}
	return false
}

func (g GlassBottle) EncodeItem() (name string, meta int16) {
	return "minecraft:glass_bottle", 0
}
