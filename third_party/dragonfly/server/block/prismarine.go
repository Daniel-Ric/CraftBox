package block

import (
	"github.com/df-mc/dragonfly/server/world"
)

type Prismarine struct {
	solid
	bassDrum

	Type PrismarineType
}

func (p Prismarine) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(p)).withBlastResistance(6)
}

func (p Prismarine) EncodeItem() (id string, meta int16) {
	return "minecraft:" + p.Type.String(), 0
}

func (p Prismarine) EncodeBlock() (name string, properties map[string]any) {
	return "minecraft:" + p.Type.String(), nil
}

func allPrismarine() (c []world.Block) {
	for _, t := range PrismarineTypes() {
		c = append(c, Prismarine{Type: t})
	}
	return
}
