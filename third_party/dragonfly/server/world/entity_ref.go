package world

import (
	"context"
	"fmt"
	"time"
)

type EntityRef[T Entity] struct {
	h *EntityHandle
}

func NewEntityRef[T Entity](h *EntityHandle) EntityRef[T] { return EntityRef[T]{h: h} }

func (r EntityRef[T]) Handle() *EntityHandle { return r.h }

func (r EntityRef[T]) Do(f func(tx *Tx, e T)) *Task {
	return r.h.schedule(typed(f))
}

func (r EntityRef[T]) DoAfter(delay time.Duration, f func(tx *Tx, e T)) *Task {
	return r.h.scheduleAfter(delay, typed(f))
}

func typed[T Entity](f func(tx *Tx, e T)) func(*Tx, Entity) error {
	return func(tx *Tx, e Entity) error {
		v, err := assertEntity[T](e)
		if err != nil {
			return err
		}
		f(tx, v)
		return nil
	}
}

func CallRef[T any, E Entity](ctx context.Context, ref EntityRef[E], f func(tx *Tx, e E) (T, error)) (T, error) {
	var zero T
	ctx, err := callContext(ctx)
	if err != nil {
		return zero, err
	}
	var result T
	task := ref.h.schedule(func(tx *Tx, e Entity) error {
		v, err := assertEntity[E](e)
		if err != nil {
			return err
		}
		var callErr error
		result, callErr = f(tx, v)
		return callErr
	})
	return awaitTask(ctx, task, &result)
}

func assertEntity[T Entity](e Entity) (T, error) {
	v, ok := e.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("%w: got %T", ErrEntityType, e)
	}
	return v, nil
}
