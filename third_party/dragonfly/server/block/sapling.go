package block

import (
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type Sapling struct {
	empty
	transparent

	Type SaplingType

	Aged bool
}

var (
	_ item.BoneMealAffected = Sapling{}
	_ Flammable             = Sapling{}
)

func (s Sapling) BoneMeal(pos cube.Pos, tx *world.Tx) item.BoneMealResult {
	if !s.growable(pos, tx) {
		return item.BoneMealResultNone
	}
	if rand.Float64() < 0.45 && !s.grow(pos, tx) {
		return item.BoneMealResultNone
	}
	return item.BoneMealResultSmall
}

func (s Sapling) FlammabilityInfo() FlammabilityInfo {
	return newFlammabilityInfo(30, 60, true)
}

func (s Sapling) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if !supportsVegetation(s, tx.Block(pos.Side(cube.FaceDown))) {
		breakBlock(s, pos, tx)
	}
}

func (s Sapling) RandomTick(pos cube.Pos, tx *world.Tx, r *rand.Rand) {
	if tx.Light(pos) >= 8 && r.IntN(7) == 0 {
		_ = s.grow(pos, tx)
	}
}

func (s Sapling) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, s)
	if !used || !supportsVegetation(s, tx.Block(pos.Side(cube.FaceDown))) {
		return false
	}
	place(tx, pos, Sapling{Type: s.Type}, user, ctx)
	return placed(ctx)
}

func (s Sapling) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(Sapling{Type: s.Type}))
}

func (Sapling) FuelInfo() item.FuelInfo {
	return newFuelInfo(time.Second * 5)
}

func (Sapling) CompostChance() float64 {
	return 0.3
}

func (Sapling) HasLiquidDrops() bool {
	return true
}

func (s Sapling) EncodeItem() (name string, meta int16) {
	return "minecraft:" + s.Type.String(), 0
}

func (s Sapling) EncodeBlock() (name string, properties map[string]any) {
	return "minecraft:" + s.Type.String(), map[string]any{"age_bit": s.Aged}
}

func (s Sapling) growable(pos cube.Pos, tx *world.Tx) bool {
	switch s.Type {
	case DarkOakSapling(), PaleOakSapling():
		_, ok := saplingSquare(pos, tx, s.Type)
		return ok
	}
	return true
}

func (s Sapling) grow(pos cube.Pos, tx *world.Tx) bool {
	if !s.Aged {
		s.Aged = true
		tx.SetBlock(pos, s, nil)
		return true
	}
	return growTree(s.Type, pos, tx)
}

func allSaplings() (saplings []world.Block) {
	for _, t := range SaplingTypes() {
		saplings = append(saplings, Sapling{Type: t}, Sapling{Type: t, Aged: true})
	}
	return
}
