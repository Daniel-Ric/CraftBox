package block

import "github.com/df-mc/dragonfly/server/item"

type PolishedBlackstoneBrick struct {
	solid
	bassDrum

	Cracked bool
}

func (b PolishedBlackstoneBrick) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, oneOf(b)).withBlastResistance(6)
}

func (b PolishedBlackstoneBrick) SmeltInfo() item.SmeltInfo {
	if b.Cracked {
		return item.SmeltInfo{}
	}
	return newSmeltInfo(item.NewStack(PolishedBlackstoneBrick{Cracked: true}, 1), 0.1)
}

func (b PolishedBlackstoneBrick) EncodeItem() (name string, meta int16) {
	name = "polished_blackstone_bricks"
	if b.Cracked {
		name = "cracked_" + name
	}
	return "minecraft:" + name, 0
}

func (b PolishedBlackstoneBrick) EncodeBlock() (string, map[string]any) {
	name := "polished_blackstone_bricks"
	if b.Cracked {
		name = "cracked_" + name
	}
	return "minecraft:" + name, nil
}
