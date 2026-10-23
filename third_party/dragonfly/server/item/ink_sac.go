package item

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type InkSac struct {
	Glowing bool
}

func (i InkSac) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, user User, ctx *UseContext) bool {
	if in, ok := tx.Block(pos).(inkable); ok {
		if res, ok := in.Ink(pos, user.Position(), i.Glowing); ok {
			tx.SetBlock(pos, res, nil)
			ctx.SubtractFromCount(1)
			return true
		}
	}
	return false
}

type inkable interface {
	Ink(pos cube.Pos, userPos mgl64.Vec3, glowing bool) (world.Block, bool)
}

func (i InkSac) EncodeItem() (name string, meta int16) {
	if i.Glowing {
		return "minecraft:glow_ink_sac", 0
	}
	return "minecraft:ink_sac", 0
}
