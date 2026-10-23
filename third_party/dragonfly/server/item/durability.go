package item

type Durable interface {
	DurabilityInfo() DurabilityInfo
}

type DurabilityInfo struct {
	MaxDurability int

	BrokenItem func() Stack

	AttackDurability, BreakDurability int

	Persistent bool
}

type Repairable interface {
	Durable
	RepairableBy(i Stack) bool
}

func simpleItem(i Stack) func() Stack {
	return func() Stack {
		return i
	}
}
