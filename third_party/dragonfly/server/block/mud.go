package block

import "github.com/df-mc/dragonfly/server/world"

type Mud struct {
	solid
}

func (Mud) SoilFor(block world.Block) bool {
	switch block.(type) {
	case ShortGrass, Fern, DoubleTallGrass, Flower, DoubleFlower, NetherSprouts, PinkPetals, DeadBush, BambooSapling, Bamboo, Sapling, RedShrub:
		return true
	}
	return false
}

func (m Mud) BreakInfo() BreakInfo {
	return newBreakInfo(0.5, alwaysHarvestable, shovelEffective, oneOf(m))
}

func (Mud) EncodeItem() (name string, meta int16) {
	return "minecraft:mud", 0
}

func (Mud) EncodeBlock() (string, map[string]any) {
	return "minecraft:mud", nil
}
