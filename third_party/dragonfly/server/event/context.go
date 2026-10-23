package event

type Context[T any] struct {
	cancel bool
	val    T
}

func C[T any](v T) *Context[T] {
	return &Context[T]{val: v}
}

func (ctx *Context[T]) Val() T {
	return ctx.val
}

func (ctx *Context[T]) Cancelled() bool {
	return ctx.cancel
}

func (ctx *Context[T]) Cancel() {
	ctx.cancel = true
}
