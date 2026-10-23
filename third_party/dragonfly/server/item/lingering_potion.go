package item

import (
	"github.com/df-mc/dragonfly/server/item/potion"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
)

type LingeringPotion struct {
	Type potion.Potion
}

func (l LingeringPotion) MaxCount() int {
	return 1
}

func (l LingeringPotion) Use(tx *world.Tx, user User, ctx *UseContext) bool {
	create := tx.World().EntityRegistry().Config().LingeringPotion
	opts := world.EntitySpawnOpts{Position: eyePosition(user), Velocity: throwableOffset(user.Rotation()).Vec3().Mul(0.5)}
	tx.AddEntity(create(opts, l.Type, user))
	tx.PlaySound(user.Position(), sound.ItemThrow{})

	ctx.SubtractFromCount(1)
	return true
}

func (l LingeringPotion) EncodeItem() (name string, meta int16) {
	return "minecraft:lingering_potion", int16(l.Type.Uint8())
}
