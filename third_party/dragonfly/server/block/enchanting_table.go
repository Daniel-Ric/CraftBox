package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

type EnchantingTable struct {
	transparent
	bassDrum
	sourceWaterDisplacer
}

func (e EnchantingTable) Model() world.BlockModel {
	return model.EnchantingTable{}
}

func (e EnchantingTable) BreakInfo() BreakInfo {
	return newBreakInfo(5, pickaxeHarvestable, pickaxeEffective, oneOf(e)).withBlastResistance(1200)
}

func (EnchantingTable) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func (EnchantingTable) LightEmissionLevel() uint8 {
	return 7
}

func (EnchantingTable) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	if opener, ok := u.(ContainerOpener); ok {
		opener.OpenBlockContainer(pos, tx)
		return true
	}
	return false
}

func (EnchantingTable) EncodeItem() (name string, meta int16) {
	return "minecraft:enchanting_table", 0
}

func (EnchantingTable) EncodeBlock() (string, map[string]any) {
	return "minecraft:enchanting_table", nil
}

func (e EnchantingTable) EncodeNBT() map[string]any {
	return map[string]any{"id": "EnchantTable"}
}

func (e EnchantingTable) DecodeNBT(map[string]any) any {
	return e
}
