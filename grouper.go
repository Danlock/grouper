// Package grouper provides a generic form of golang.og//sync/errgroup.
// Can support generic values being returned from functions as well as errors.
package grouper

import (
	"context"
	"sync"
)

// A Group is a collection of goroutines working on subtasks that are part of
// the same overall task.
type Group[V any] struct {
	ctx    context.Context
	cancel func()

	wg sync.WaitGroup

	errOnce sync.Once
	err     error

	funcs []func(context.Context) (V, error)
}

// New returns a new Group and an associated Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func New[V any](ctx context.Context, funcs ...func(context.Context) (V, error)) (*Group[V], context.Context) {
	if len(funcs) == 0 {
		panic("grouper.New cannot be created without funcs to call!")
	}
	g := &Group[V]{funcs: funcs}
	g.ctx, g.cancel = context.WithCancel(ctx)
	g.wg.Add(len(funcs))
	return g, g.ctx
}

// Wait blocks until all function calls passed into New have returned, then
// returns the successfully returned values and the first non-nil error (if any).
// Note that the values slice could be sparse in the case of multiple errors.
func (g *Group[V]) Wait() ([]V, error) {
	values := make([]V, len(g.funcs))
	for i, f := range g.funcs {
		i, f := i, f
		go func() {
			defer g.wg.Done()

			if v, err := f(g.ctx); err != nil {
				g.errOnce.Do(func() {
					g.err = err
					if g.cancel != nil {
						g.cancel()
					}
				})
			} else {
				// Use the index to store the value into the slice element that only this goroutine will mutate
				values[i] = v
			}
		}()
	}

	g.wg.Wait()
	g.cancel()
	return values, g.err
}
