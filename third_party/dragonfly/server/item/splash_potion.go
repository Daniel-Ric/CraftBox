package item

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item/potion"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"math"
)

type SplashPotion struct {
	Type potion.Potion
}

func (s SplashPotion) MaxCount() int {
	return 1
}

func (s SplashPotion) Use(tx *world.Tx, user User, ctx *UseContext) bool {
	create := tx.World().EntityRegistry().Config().SplashPotion
	opts := world.EntitySpawnOpts{Position: eyePosition(user), Velocity: throwableOffset(user.Rotation()).Vec3().Mul(0.5)}
	tx.AddEntity(create(opts, s.Type, user))
	tx.PlaySound(user.Position(), sound.ItemThrow{})

	ctx.SubtractFromCount(1)
	return true
}

func throwableOffset(r cube.Rotation) cube.Rotation {
	r[1] = max(min(r[1], 89.9), -89.9)

	r[1] -= math.Sqrt(max(0, 89.9*89.9-r[1]*r[1])) * (26.5 / 89.9)
	r[1] = max(min(r[1], 89.9), -89.9)

	return r
}

func (s SplashPotion) EncodeItem() (name string, meta int16) {
	return "minecraft:splash_potion", int16(s.Type.Uint8())
}
