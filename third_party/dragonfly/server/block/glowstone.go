package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
)

type Glowstone struct {
	solid
}

func (g Glowstone) Instrument() sound.Instrument {
	return sound.Pling()
}

func (Glowstone) CanRedstoneWireStepDown(cube.Pos, cube.Pos, world.BlockSource) bool {
	return false
}

func (Glowstone) RedstoneNonConductive() {}

func (g Glowstone) BreakInfo() BreakInfo {
	return newBreakInfo(0.3, alwaysHarvestable, nothingEffective, discreteDrops(item.GlowstoneDust{}, g, 2, 4, 4))
}

func (Glowstone) EncodeItem() (name string, meta int16) {
	return "minecraft:glowstone", 0
}

func (Glowstone) EncodeBlock() (string, map[string]any) {
	return "minecraft:glowstone", nil
}

func (Glowstone) LightEmissionLevel() uint8 {
	return 15
}
