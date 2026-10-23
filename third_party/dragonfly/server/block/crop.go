package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Crop interface {
	GrowthStage() int

	SameCrop(c Crop) bool
}

type crop struct {
	transparent
	empty

	Growth int
}

func (c crop) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if _, ok := tx.Block(pos.Side(cube.FaceDown)).(Farmland); !ok {
		breakBlock(tx.Block(pos), pos, tx)
	}
}

func (c crop) HasLiquidDrops() bool {
	return true
}

func (c crop) GrowthStage() int {
	return c.Growth
}

func (c crop) CalculateGrowthChance(pos cube.Pos, tx *world.Tx) float64 {
	points := 0.0

	block := tx.Block(pos)
	under := pos.Side(cube.FaceDown)

	for x := -1; x <= 1; x++ {
		for z := -1; z <= 1; z++ {
			block := tx.Block(under.Add(cube.Pos{x, 0, z}))
			if farmland, ok := block.(Farmland); ok {
				farmlandPoints := 0.0
				if farmland.Hydration > 0 {
					farmlandPoints = 4
				} else {
					farmlandPoints = 2
				}
				if x != 0 || z != 0 {
					farmlandPoints = (farmlandPoints - 1) / 4
				}
				points += farmlandPoints
			}
		}
	}

	north := pos.Side(cube.FaceNorth)
	south := pos.Side(cube.FaceSouth)

	northSouth := sameCrop(block, tx.Block(north)) || sameCrop(block, tx.Block(south))
	westEast := sameCrop(block, tx.Block(pos.Side(cube.FaceWest))) || sameCrop(block, tx.Block(pos.Side(cube.FaceEast)))
	if northSouth && westEast {
		points /= 2
	} else {
		diagonal := sameCrop(block, tx.Block(north.Side(cube.FaceWest))) ||
			sameCrop(block, tx.Block(north.Side(cube.FaceEast))) ||
			sameCrop(block, tx.Block(south.Side(cube.FaceWest))) ||
			sameCrop(block, tx.Block(south.Side(cube.FaceEast)))
		if diagonal {
			points /= 2
		}
	}

	chance := 1 / (25/points + 1)
	return chance
}

func sameCrop(blockA, blockB world.Block) bool {
	if a, ok := blockA.(Crop); ok {
		if b, ok := blockB.(Crop); ok {
			return a.SameCrop(b)
		}
	}
	return false
}
