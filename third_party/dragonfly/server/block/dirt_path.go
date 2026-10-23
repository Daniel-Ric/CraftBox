package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type DirtPath struct {
	tilledGrass
	transparent
}

func (p DirtPath) Till() (world.Block, bool) {
	return Farmland{}, true
}

func (p DirtPath) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	up := pos.Side(cube.FaceUp)
	if tx.Block(up).Model().FaceSolid(up, cube.FaceDown, tx) {

		tx.SetBlock(pos, Dirt{}, nil)
	}
}

func (p DirtPath) BreakInfo() BreakInfo {
	return newBreakInfo(0.65, alwaysHarvestable, shovelEffective, silkTouchOneOf(Dirt{}, p))
}

func (DirtPath) EncodeItem() (name string, meta int16) {
	return "minecraft:grass_path", 0
}

func (DirtPath) EncodeBlock() (string, map[string]any) {
	return "minecraft:grass_path", nil
}
