package goworker

import (
	"context"
	"sync"
)

// errGroup A Group is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero Group is valid and does not cancel on error.
type errGroup struct {
	cancel func()

	wg sync.WaitGroup

	errOnce sync.Once
	err     error
}

// errGroupWithContext returns a new Group and an associated Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to goWork
// returns a non-nil error or the first time wait returns, whichever occurs
// first.
func errGroupWithContext(ctx context.Context) (*errGroup, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &errGroup{cancel: cancel}, ctx
}

// wait blocks until all function calls from the goWork method have returned, then
// returns the first non-nil error (if any) from them.
func (g *errGroup) wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}

// goWork calls the given function in a new goroutine.
//
// The first call to return a non-nil error cancels the group; its error will be
// returned by wait.
func (g *errGroup) goWork(f func(*Worker) error, ig *Worker) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		if err := f(ig); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
	}()
}
