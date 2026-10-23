package inventory

import (
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
)

type Holder interface{}

type Context = event.Context[Holder]

type Handler interface {
	HandleTake(ctx *Context, slot int, it item.Stack)

	HandlePlace(ctx *Context, slot int, it item.Stack)

	HandleDrop(ctx *Context, slot int, it item.Stack)
}

var _ Handler = NopHandler{}

type NopHandler struct{}

func (NopHandler) HandleTake(*Context, int, item.Stack)  {}
func (NopHandler) HandlePlace(*Context, int, item.Stack) {}
func (NopHandler) HandleDrop(*Context, int, item.Stack)  {}
