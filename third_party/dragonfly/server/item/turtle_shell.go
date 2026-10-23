package item

import "github.com/df-mc/dragonfly/server/world"

type TurtleShell struct{}

func (TurtleShell) Use(_ *world.Tx, _ User, ctx *UseContext) bool {
	ctx.SwapHeldWithArmour(0)
	return false
}

func (TurtleShell) DurabilityInfo() DurabilityInfo {
	return DurabilityInfo{
		MaxDurability: 276,
		BrokenItem:    simpleItem(Stack{}),
	}
}

func (TurtleShell) RepairableBy(i Stack) bool {
	_, ok := i.Item().(Scute)
	return ok
}

func (TurtleShell) MaxCount() int {
	return 1
}

func (TurtleShell) DefencePoints() float64 {
	return 2
}

func (TurtleShell) Toughness() float64 {
	return 0
}

func (TurtleShell) KnockBackResistance() float64 {
	return 0
}

func (TurtleShell) EnchantmentValue() int {
	return 9
}

func (TurtleShell) Helmet() bool {
	return true
}

func (TurtleShell) EncodeItem() (name string, meta int16) {
	return "minecraft:turtle_helmet", 0
}
