package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

type SmithingTable struct {
	bass
	solid
}

func (SmithingTable) EncodeItem() (name string, meta int16) {
	return "minecraft:smithing_table", 0
}

func (SmithingTable) EncodeBlock() (name string, properties map[string]interface{}) {
	return "minecraft:smithing_table", nil
}

func (s SmithingTable) BreakInfo() BreakInfo {
	return newBreakInfo(2.5, alwaysHarvestable, axeEffective, oneOf(s))
}

func (SmithingTable) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	if opener, ok := u.(ContainerOpener); ok {
		opener.OpenBlockContainer(pos, tx)
		return true
	}
	return false
}
