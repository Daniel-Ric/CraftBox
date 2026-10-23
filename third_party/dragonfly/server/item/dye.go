package item

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type Dye struct {
	Colour Colour
}

func (d Dye) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, user User, ctx *UseContext) bool {
	if dy, ok := tx.Block(pos).(dyeable); ok {
		if res, ok := dy.Dye(pos, user.Position(), d.Colour); ok {
			tx.SetBlock(pos, res, nil)
			ctx.SubtractFromCount(1)
			return true
		}
	}
	return false
}

type dyeable interface {
	Dye(pos cube.Pos, userPos mgl64.Vec3, c Colour) (world.Block, bool)
}

func (d Dye) EncodeItem() (name string, meta int16) {
	return "minecraft:" + d.Colour.String() + "_dye", 0
}
