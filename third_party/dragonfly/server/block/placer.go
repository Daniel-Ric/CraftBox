package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

type Placer interface {
	item.User
	PlaceBlock(pos cube.Pos, b world.Block, ctx *item.UseContext)
}
