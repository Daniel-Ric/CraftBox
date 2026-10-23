package block

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Dirt struct {
	solid

	Coarse bool
}

func (d Dirt) SoilFor(block world.Block) bool {
	switch block.(type) {
	case ShortGrass, Fern, DoubleTallGrass, DeadBush, RedShrub:
		return !d.Coarse
	case Flower, DoubleFlower, NetherSprouts, PinkPetals, SugarCane, BambooSapling, Bamboo, Sapling:
		return true
	}
	return false
}

func (d Dirt) BreakInfo() BreakInfo {
	return newBreakInfo(0.5, alwaysHarvestable, shovelEffective, oneOf(d))
}

func (d Dirt) Till() (world.Block, bool) {
	if d.Coarse {
		return Dirt{Coarse: false}, true
	}
	return Farmland{}, true
}

func (d Dirt) Shovel() (world.Block, bool) {
	return DirtPath{}, true
}

func (d Dirt) EncodeItem() (name string, meta int16) {
	if d.Coarse {
		return "minecraft:coarse_dirt", 0
	}
	return "minecraft:dirt", 0
}

func (d Dirt) EncodeBlock() (string, map[string]any) {
	if d.Coarse {
		return "minecraft:coarse_dirt", nil
	}
	return "minecraft:dirt", nil
}

func spreadDirt(b world.Block, pos cube.Pos, tx *world.Tx, r *rand.Rand, minY int) {
	aboveLight := tx.Light(pos.Side(cube.FaceUp))
	if aboveLight < 4 {
		tx.SetBlock(pos, Dirt{}, nil)
		return
	}
	if aboveLight < 9 {
		return
	}

	n := r.Uint32()

	for range 4 {
		x, y, z := int(n)%3, int(n>>2)%5, int(n>>5)%3
		n >>= 7

		spreadPos := pos.Add(cube.Pos{x - 1, y + minY, z - 1})
		if tx.Light(spreadPos.Side(cube.FaceUp)) < 4 {
			continue
		}
		if dirt, ok := tx.Block(spreadPos).(Dirt); !ok || dirt.Coarse {
			continue
		}
		tx.SetBlock(spreadPos, b, nil)
	}
}

func supportsVegetation(vegetation, block world.Block) bool {
	soil, ok := block.(Soil)
	return ok && soil.SoilFor(vegetation)
}

type Soil interface {
	SoilFor(world.Block) bool
}
