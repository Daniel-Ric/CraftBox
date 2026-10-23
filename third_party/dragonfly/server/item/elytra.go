package item

import "github.com/df-mc/dragonfly/server/world"

type Elytra struct{}

func (Elytra) Use(_ *world.Tx, _ User, ctx *UseContext) bool {
	ctx.SwapHeldWithArmour(1)
	return false
}

func (Elytra) DurabilityInfo() DurabilityInfo {
	return DurabilityInfo{
		MaxDurability: 433,
		Persistent:    true,
		BrokenItem:    simpleItem(Stack{}),
	}
}

func (Elytra) RepairableBy(i Stack) bool {
	_, ok := i.Item().(PhantomMembrane)
	return ok
}

func (Elytra) MaxCount() int {
	return 1
}

func (Elytra) Chestplate() bool {
	return true
}

func (Elytra) DefencePoints() float64 {
	return 0
}

func (e Elytra) Toughness() float64 {
	return 0
}

func (e Elytra) KnockBackResistance() float64 {
	return 0
}

func (Elytra) EncodeItem() (name string, meta int16) {
	return "minecraft:elytra", 0
}
