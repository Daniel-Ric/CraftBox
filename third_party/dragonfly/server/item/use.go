package item

type UseContext struct {
	Damage int

	CountSub int

	IgnoreBBox bool

	NewItem Stack

	ConsumedItems []Stack

	NewItemSurvivalOnly bool

	FirstFunc func(comparable func(Stack) bool) (Stack, bool)

	SwapHeldWithArmour func(i int)
}

func (ctx *UseContext) Consume(s Stack) {
	ctx.ConsumedItems = append(ctx.ConsumedItems, s)
}

func (ctx *UseContext) DamageItem(d int) { ctx.Damage += d }

func (ctx *UseContext) SubtractFromCount(d int) { ctx.CountSub += d }
