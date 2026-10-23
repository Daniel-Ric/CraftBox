package block

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

type Sandstone struct {
	solid
	bassDrum

	Type SandstoneType

	Red bool
}

func (s Sandstone) BreakInfo() BreakInfo {
	if s.Type == SmoothSandstone() {
		return newBreakInfo(2, pickaxeHarvestable, pickaxeEffective, oneOf(s)).withBlastResistance(6)
	}
	return newBreakInfo(0.8, pickaxeHarvestable, pickaxeEffective, oneOf(s))
}

func (s Sandstone) EncodeItem() (name string, meta int16) {
	var prefix string
	if s.Type != NormalSandstone() {
		prefix = s.Type.String() + "_"
	}
	if s.Red {
		return "minecraft:" + prefix + "red_sandstone", 0
	}
	return "minecraft:" + prefix + "sandstone", 0
}

func (s Sandstone) EncodeBlock() (string, map[string]any) {
	var prefix string
	if s.Type != NormalSandstone() {
		prefix = s.Type.String() + "_"
	}
	if s.Red {
		return "minecraft:" + prefix + "red_sandstone", nil
	}
	return "minecraft:" + prefix + "sandstone", nil
}

func (s Sandstone) SmeltInfo() item.SmeltInfo {
	if s.Type == NormalSandstone() {
		return newSmeltInfo(item.NewStack(Sandstone{Red: s.Red, Type: SmoothSandstone()}, 1), 0.1)
	}
	return item.SmeltInfo{}
}

func allSandstones() (c []world.Block) {
	f := func(red bool) {
		for _, t := range SandstoneTypes() {
			c = append(c, Sandstone{Type: t, Red: red})
		}
	}
	f(true)
	f(false)
	return
}
