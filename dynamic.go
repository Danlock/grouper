// Package grouper provides a generic form of golang.og//sync/errgroup.
// Can support generic values being returned from functions as well as errors.
package grouper

import (
	"context"
	"sync"
)

// A DynamicGroup is a collection of goroutines working on subtasks that are part of
// the same overall task. It can have an unbounded number of goroutines.
type DynamicGroup[V any] struct {
	ctx    context.Context
	cancel func()

	errOnce sync.Once
	err     error

	count     uint
	valueChan chan V
}

// NewDynamic returns a new DynamicGroup and an associated Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func NewDynamic[V any](ctx context.Context) (*DynamicGroup[V], context.Context) {
	g := &DynamicGroup[V]{valueChan: make(chan V)}
	g.ctx, g.cancel = context.WithCancel(ctx)
	return g, g.ctx
}

// Wait blocks until all function calls passed into New have returned, then
// returns the successfully returned values and the first non-nil error (if any).
// Note that the values slice contains the results of every function call, whether it errored or not. It can be sparse in the case of multiple errors.
// Wait should only be called once per DynamicGroup.
func (g *DynamicGroup[V]) Wait() ([]V, error) {
	values := make([]V, g.count)
	for i := range values {
		values[i] = <-g.valueChan
	}

	g.count = 0
	g.cancel()
	return values, g.err
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error cancels the group; its error will be
// returned by Wait.
// Go should not be called after Wait has been called.
func (g *DynamicGroup[V]) Go(f func(context.Context) (V, error)) {
	g.count++
	go func() {
		v, err := f(g.ctx)
		if err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
		g.valueChan <- v
	}()
}
