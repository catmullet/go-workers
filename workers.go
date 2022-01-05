package gorkers

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
)

// WorkFunc get input and could put in outChan for followers.
// ⚠️ outChan could be closed if follower is stoped before producer.
// error returned can be process by afterFunc but will be ignored by default.
type WorkFunc func(ctx context.Context, in interface{}, out chan<- interface{}) error

type BeforeFunc func(ctx context.Context) error
type AfterFunc func(ctx context.Context, in interface{}, err error) error

type Runner struct {
	ctx         context.Context
	cancel      context.CancelFunc
	inputCtx    context.Context
	inputCancel context.CancelFunc
	inChan      chan interface{}
	outChan     chan interface{}

	afterFunc  AfterFunc
	workFunc   WorkFunc
	beforeFunc BeforeFunc

	timeout time.Duration

	maxWorkers int64
	started    *sync.Once
	done       chan struct{}

	metricSend uint32
	metricOK   uint32
	metricFail uint32
}

// NewRunner Factory function for a new Runner.  The Runner will handle running the workers logic.
func NewRunner(ctx context.Context, w WorkFunc, maxWorkers, buffer int64) *Runner {
	runnerCtx, runnerCancel := context.WithCancel(ctx)
	inputCtx, inputCancel := context.WithCancel(runnerCtx)

	runner := &Runner{
		ctx:         runnerCtx,
		cancel:      runnerCancel,
		inputCtx:    inputCtx,
		inputCancel: inputCancel,
		inChan:      make(chan interface{}, buffer),
		outChan:     nil,
		afterFunc:   func(ctx context.Context, in interface{}, err error) error { return nil },
		workFunc:    w,
		beforeFunc:  func(ctx context.Context) error { return nil },
		maxWorkers:  maxWorkers,
		started:     new(sync.Once),
		done:        make(chan struct{}),
	}
	return runner
}

var ErrInputClosed = errors.New("input closed")

// Send Send an object to the worker for processing if context is not Done.
func (r *Runner) Send(in interface{}) error {
	select {
	case <-r.inputCtx.Done():
		return ErrInputClosed
	case r.inChan <- in:
		atomic.AddUint32(&r.metricSend, uint32(1))
	}
	return nil
}

// InFrom Set a worker to accept output from another worker(s).
func (r *Runner) InFrom(w ...*Runner) *Runner {
	for _, wr := range w {
		// in := make(chan interface{})
		// go func(in chan interface{}) {
		// 	for msg := range in {
		// 		if err := r.Send(msg); err != nil {
		// 			return
		// 		}
		// 	}
		// }(in)
		wr.SetOut(r.inChan) // nolint
	}
	return r
}

// Start execute beforeFunc and launch worker processing.
func (r *Runner) Start() error {
	r.started.Do(func() {
		if err := r.beforeFunc(r.ctx); err == nil {
			go r.work()
		}
	})
	return nil
}

// BeforeFunc Function to be run before worker starts processing.
func (r *Runner) BeforeFunc(f BeforeFunc) *Runner {
	r.beforeFunc = f
	return r
}

// AfterFunc Function to be run after worker has stopped.
// It can be used for logging and error management.
// input can be retreive with context value:
//   ctx.Value(workers.InputKey{})
// ⚠️ If an error is returned it stop Runner execution.
func (r *Runner) AfterFunc(f AfterFunc) *Runner {
	r.afterFunc = f
	return r
}

var ErrOutAlready = errors.New("out already set")

// SetOut Allows the setting of a workers out channel, if not already set.
func (r *Runner) SetOut(c chan interface{}) error {
	if r.outChan != nil {
		return ErrOutAlready
	}
	r.outChan = c
	return nil
}

// SetDeadline allows a time to be set when the Runner should stop.
// ⚠️ Should only be called before Start
func (r *Runner) SetDeadline(t time.Time) *Runner {
	r.ctx, r.cancel = context.WithDeadline(r.ctx, t)
	return r
}

// SetWorkerTimeout allows a time duration to be set when the workers should stop.
// ⚠️ Should only be called before Start
func (r *Runner) SetWorkerTimeout(duration time.Duration) *Runner {
	r.timeout = duration
	return r
}

// Wait close the input channel and waits it to drain and process.
func (r *Runner) Wait() *Runner {
	if r.inputCtx.Err() == nil {
		r.inputCancel()
		close(r.inChan)
	}

	<-r.done

	return r
}

// Stop Stops the processing of a worker and waits for workers to finish.
func (r *Runner) Stop() *Runner {
	r.cancel()
	r.Wait()
	return r
}

// work starts processing input and limits worker instance number.
func (r *Runner) work() {
	var wg sync.WaitGroup

	sem := semaphore.NewWeighted(r.maxWorkers)

	defer func() {
		wg.Wait()
		r.cancel()
		close(r.done)
	}()

	for {
		if err := sem.Acquire(r.ctx, 1); err != nil {
			return
		}

		// slot available for worker
		select {
		case <-r.ctx.Done():
			return
		case input, open := <-r.inChan:
			if !open {
				return
			}
			wg.Add(1)

			workCtx, workCancel := context.WithCancel(r.ctx)
			if r.timeout > 0 {
				workCtx, workCancel = context.WithTimeout(r.ctx, r.timeout)
			}

			go func() {
				defer func() {
					workCancel()
					sem.Release(1)
					wg.Done()
				}()
				err := r.workFunc(workCtx, input, r.outChan)
				if err == nil {
					atomic.AddUint32(&r.metricOK, uint32(1))
				} else {
					atomic.AddUint32(&r.metricFail, uint32(1))
				}
				if err := r.afterFunc(workCtx, input, err); err != nil {
					r.cancel()
				}
			}()
		}
	}
}

func (r *Runner) Metrics() (send, ok, fail uint32) {
	return r.metricSend, r.metricOK, r.metricFail
}

func (r *Runner) Fail() bool {
	return r.metricFail != 0
}
