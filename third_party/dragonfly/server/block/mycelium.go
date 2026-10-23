package block

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Mycelium struct {
	solid
}

func (m Mycelium) SoilFor(block world.Block) bool {
	switch block.(type) {
	case ShortGrass, Fern, DoubleTallGrass, Flower, DoubleFlower, NetherSprouts, PinkPetals, SugarCane, DeadBush, BambooSapling, Bamboo, RedShrub:
		return true
	}
	return false
}

func (m Mycelium) RandomTick(pos cube.Pos, tx *world.Tx, r *rand.Rand) {
	spreadDirt(m, pos, tx, r, -2)
}

func (Mycelium) Shovel() (world.Block, bool) {
	return DirtPath{}, true
}

func (m Mycelium) BreakInfo() BreakInfo {
	return newBreakInfo(0.6, alwaysHarvestable, shovelEffective, silkTouchOneOf(Dirt{}, m))
}

func (Mycelium) EncodeItem() (name string, meta int16) {
	return "minecraft:mycelium", 0
}

func (Mycelium) EncodeBlock() (string, map[string]any) {
	return "minecraft:mycelium", nil
}
