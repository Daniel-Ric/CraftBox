package item

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/go-gl/mathgl/mgl64"
)

type BoneMealResult int

const (
	BoneMealResultNone BoneMealResult = iota

	BoneMealResultSmall

	BoneMealResultArea
)

type BoneMeal struct{}

type BoneMealAffected interface {
	BoneMeal(pos cube.Pos, tx *world.Tx) BoneMealResult
}

func (b BoneMeal) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, _ User, ctx *UseContext) bool {
	if bm, ok := tx.Block(pos).(BoneMealAffected); ok {
		result := bm.BoneMeal(pos, tx)
		if result == BoneMealResultNone {
			return false
		}

		ctx.SubtractFromCount(1)
		tx.AddParticle(pos.Vec3(), particle.BoneMeal{
			Area: result == BoneMealResultArea,
		})
		return true
	}
	return false
}

func (b BoneMeal) EncodeItem() (name string, meta int16) {
	return "minecraft:bone_meal", 0
}
