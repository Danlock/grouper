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
func New[V any](funcs ...func(context.Context) (V, error)) *Group[V] {
	if len(funcs) == 0 {
		panic("grouper.New cannot be created without funcs to call!")
	}
	return &Group[V]{funcs: funcs}
}

// Wait blocks until all function calls passed into New have returned, then
// returns the successfully returned values and the first non-nil error (if any).
// Finally a Wait that returned an error will continue returning that same error on all future calls.
func (g *Group[V]) Wait(ctx context.Context) ([]V, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	g.wg.Add(len(g.funcs))

	values := make([]V, len(g.funcs))
	for i, f := range g.funcs {
		i, f := i, f
		go func() {
			defer g.wg.Done()
			v, err := f(ctx)
			if err != nil {
				g.errOnce.Do(func() {
					g.err = err
					cancel()
				})
			}
			// Use the index to store the value into the slice element that only this goroutine will mutate
			values[i] = v
		}()
	}

	g.wg.Wait()
	return values, g.err
}
