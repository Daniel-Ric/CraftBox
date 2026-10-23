package block

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type RedShrub struct {
	replaceable
	transparent
	empty
}

func (RedShrub) FlammabilityInfo() FlammabilityInfo {
	return newFlammabilityInfo(60, 100, false)
}

func (r RedShrub) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(r))
}

func (r RedShrub) BoneMeal(pos cube.Pos, tx *world.Tx) item.BoneMealResult {
	for _, offset := range rand.Perm(len(shrubSpread)) {
		at := pos.Add(shrubSpread[offset])
		if !replaceableWith(tx, at, r) || !supportsVegetation(r, tx.Block(at.Side(cube.FaceDown))) {
			continue
		}
		tx.SetBlock(at, r, nil)
		return item.BoneMealResultSmall
	}
	return item.BoneMealResultNone
}

var shrubSpread = []cube.Pos{
	{1}, {-1}, {0, 0, 1}, {0, 0, -1},
	{1, 1}, {-1, 1}, {0, 1, 1}, {0, 1, -1},
	{1, -1}, {-1, -1}, {0, -1, 1}, {0, -1, -1},
}

func (RedShrub) CompostChance() float64 {
	return 0.3
}

func (RedShrub) HasLiquidDrops() bool {
	return true
}

func (r RedShrub) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if !supportsVegetation(r, tx.Block(pos.Side(cube.FaceDown))) {
		breakBlock(r, pos, tx)
	}
}

func (r RedShrub) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, r)
	if !used || !supportsVegetation(r, tx.Block(pos.Side(cube.FaceDown))) {
		return false
	}
	place(tx, pos, r, user, ctx)
	return placed(ctx)
}

func (RedShrub) EncodeItem() (name string, meta int16) {
	return "minecraft:red_shrub", 0
}

func (RedShrub) EncodeBlock() (name string, properties map[string]any) {
	return "minecraft:red_shrub", nil
}
