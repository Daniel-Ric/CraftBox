package item

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/portal"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

type EnderEye struct{}

func (EnderEye) EncodeItem() (name string, meta int16) {
	return "minecraft:ender_eye", 0
}

func (EnderEye) MaxCount() int {
	return 64
}

type endPortalFrame interface {
	InsertEndPortalEye() (world.Block, bool)
}

func (EnderEye) UseOnBlock(pos cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, _ User, ctx *UseContext) bool {
	f, ok := tx.Block(pos).(endPortalFrame)
	if !ok {
		return false
	}
	updated, inserted := f.InsertEndPortalEye()
	if !inserted {
		return false
	}
	tx.SetBlock(pos, updated, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.EnderEyePlaced{})
	if portal.ActivateEndPortal(tx, pos) {
		tx.PlaySound(pos.Vec3Centre(), sound.EndPortalCreated{})
	}
	ctx.SubtractFromCount(1)
	return true
}
