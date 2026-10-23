package item

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"time"
)

type GoatHorn struct {
	nopReleasable

	Type sound.Horn
}

func (GoatHorn) MaxCount() int {
	return 1
}

func (GoatHorn) Cooldown() time.Duration {
	return time.Second * 7
}

func (g GoatHorn) Use(tx *world.Tx, user User, _ *UseContext) bool {
	tx.PlaySound(user.Position(), sound.GoatHorn{Horn: g.Type})
	user.H().DoAfter(time.Second, g.releaseItem)
	return true
}

func (g GoatHorn) releaseItem(_ *world.Tx, e world.Entity) {
	user := e.(User)
	if !user.UsingItem() {

		return
	}
	held, _ := user.HeldItems()
	if _, ok := held.Item().(GoatHorn); !ok {

		return
	}

	user.ReleaseItem()
}

func (g GoatHorn) EncodeItem() (name string, meta int16) {
	return "minecraft:goat_horn", int16(g.Type.Uint8())
}
