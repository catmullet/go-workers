package workers

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var defaultWatchSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL}

// Worker Contains the work function. Allows an input and output to a channel or another worker for pipeline work.
// Return nil if you want the Runner to continue otherwise any error will cause the Runner to shutdown and return the
// error.
type Worker interface {
	Work(in interface{}, out chan<- interface{}) error
}

// Runner Handles the running the Worker logic.
type Runner interface {
	BeforeFunc(func(ctx context.Context) error) Runner
	AfterFunc(func(ctx context.Context, err error) error) Runner
	SetDeadline(t time.Time) Runner
	SetTimeout(duration time.Duration) Runner
	SetFollower()
	Send(in interface{})
	InFrom(w ...Runner) Runner
	SetOut(chan interface{})
	Start() Runner
	Stop() chan error
	Wait() error
	AvailableWorkers() int64
}

type runner struct {
	ctx        context.Context
	cancel     context.CancelFunc
	inChan     chan interface{}
	outChan    chan interface{}
	errChan    chan error
	signalChan chan os.Signal
	limiter    chan struct{}

	afterFunc  func(ctx context.Context, err error) error
	workFunc   func(in interface{}, out chan<- interface{}) error
	beforeFunc func(ctx context.Context) error

	timeout  time.Duration
	deadline time.Duration

	isLeader   bool
	stopCalled bool

	numWorkers int64
	lock       *sync.RWMutex
	wg         *sync.WaitGroup
	done       *sync.Once
	once       *sync.Once
}

// NewRunner Factory function for a new Runner.  The Runner will handle running the workers logic.
func NewRunner(ctx context.Context, w Worker, numWorkers int64) Runner {
	var runnerCtx, runnerCancel = context.WithCancel(ctx)
	var runner = &runner{
		ctx:        runnerCtx,
		cancel:     runnerCancel,
		inChan:     make(chan interface{}, numWorkers),
		outChan:    nil,
		errChan:    make(chan error, 1),
		signalChan: make(chan os.Signal, 1),
		limiter:    make(chan struct{}, numWorkers),
		afterFunc:  func(ctx context.Context, err error) error { return err },
		workFunc:   w.Work,
		beforeFunc: func(ctx context.Context) error { return nil },
		numWorkers: numWorkers,
		isLeader:   true,
		lock:       new(sync.RWMutex),
		wg:         new(sync.WaitGroup),
		once:       new(sync.Once),
		done:       new(sync.Once),
	}
	runner.waitForSignal(defaultWatchSignals...)
	return runner
}

// Send Send an object to the worker for processing.
func (r *runner) Send(in interface{}) {
	select {
	case <-r.ctx.Done():
		return
	case r.inChan <- in:
	}
}

// InFrom Set a worker to accept output from another worker(s).
func (r *runner) InFrom(w ...Runner) Runner {
	r.SetFollower()
	for _, wr := range w {
		wr.SetOut(r.inChan)
	}
	return r
}

// SetFollower Sets the worker as a follower and does not need to close it's in channel.
func (r *runner) SetFollower() {
	r.lock.Lock()
	r.isLeader = false
	r.lock.Unlock()
}

// Start Starts the worker on processing.
func (r *runner) Start() Runner {
	r.startWork()
	return r
}

// BeforeFunc Function to be run before worker starts processing.
func (r *runner) BeforeFunc(f func(ctx context.Context) error) Runner {
	r.beforeFunc = f
	return r
}

// AfterFunc Function to be run after worker has stopped.
func (r *runner) AfterFunc(f func(ctx context.Context, err error) error) Runner {
	r.afterFunc = f
	return r
}

// SetOut Allows the setting of a workers out channel, if not already set.
func (r *runner) SetOut(c chan interface{}) {
	if r.outChan != nil {
		return
	}
	r.outChan = c
}

// SetDeadline allows a time to be set when the workers should stop.
// Deadline needs to be handled by the IsDone method.
func (r *runner) SetDeadline(t time.Time) Runner {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.ctx, r.cancel = context.WithDeadline(r.ctx, t)
	return r
}

// SetTimeout allows a time duration to be set when the workers should stop.
// Timeout needs to be handled by the IsDone method.
func (r *runner) SetTimeout(duration time.Duration) Runner {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.timeout = duration
	return r
}

// Wait calls stop on workers and waits for the channel to drain.
// !!Should only be called when certain nothing will send to worker.
func (r *runner) Wait() error {
	r.waitForDrain()
	if err := <-r.Stop(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// Stop Stops the processing of a worker and closes it's channel in.
// Returns a blocking channel with type error.
// !!Should only be called when certain nothing will send to worker.
func (r *runner) Stop() chan error {
	r.done.Do(func() {
		if r.inChan != nil && r.isLeader {
			close(r.inChan)
		}
	})
	return r.errChan
}

// IsDone returns a channel signaling the workers context has been canceled.
func (r *runner) IsDone() <-chan struct{} {
	return r.ctx.Done()
}

// waitForSignal make sure we wait for a term signal and shutdown correctly
func (r *runner) waitForSignal(signals ...os.Signal) {
	go func() {
		signal.Notify(r.signalChan, signals...)
		<-r.signalChan
		if r.cancel != nil {
			r.cancel()
		}
	}()
}

// waitForDrain Waits for the limiter to be zeroed out and the in channel to be empty.
func (r *runner) waitForDrain() {
	for len(r.limiter) > 0 || len(r.inChan) > 0 {
		// Wait for the drain.
	}
}

// AvailableWorkers .
func (r *runner) AvailableWorkers() int64 {
	return r.numWorkers - int64(len(r.limiter))
}

// startWork Runs the before function and starts processing until one of three things happen.
// 1. A term signal is received or cancellation of context.
// 2. Stop function is called.
// 3. Worker returns an error.
func (r *runner) startWork() {
	var err error
	if err = r.beforeFunc(r.ctx); err != nil {
		r.errChan <- err
		return
	}
	if r.timeout > 0 {
		r.ctx, r.cancel = context.WithTimeout(r.ctx, r.timeout)
	}
	r.wg.Add(1)
	go func() {
		var workerWG = new(sync.WaitGroup)
		var closeOnce = new(sync.Once)

		// write out error if not nil on exit.
		defer func() {
			workerWG.Wait()
			r.errChan <- err
			closeOnce.Do(func() {
				if r.outChan != nil {
					close(r.outChan)
				}
			})
			r.wg.Done()
		}()
		for in := range r.inChan {
			input := in
			r.limiter <- struct{}{}
			workerWG.Add(1)
			go func() {
				defer func() {
					<-r.limiter
					workerWG.Done()
				}()
				if err := r.workFunc(input, r.outChan); err != nil {
					r.once.Do(func() {
						r.errChan <- err
						r.cancel()
						return
					})
				}
			}()
		}
	}()
}
