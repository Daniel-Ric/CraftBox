package item

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

type Honeycomb struct{}

func (Honeycomb) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, user User, ctx *UseContext) bool {
	if wa, ok := tx.Block(pos).(waxable); ok {
		if res, ok := wa.Wax(pos, user.Position()); ok {
			tx.SetBlock(pos, res, nil)
			tx.PlaySound(pos.Vec3(), sound.SignWaxed{})
			ctx.SubtractFromCount(1)
			return true
		}
	}
	return false
}

type waxable interface {
	Wax(pos cube.Pos, userPos mgl64.Vec3) (world.Block, bool)
}

func (Honeycomb) EncodeItem() (name string, meta int16) {
	return "minecraft:honeycomb", 0
}
