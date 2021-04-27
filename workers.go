package workers

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var defaultWatchSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL}

type Worker interface {
	Work(in interface{}, out chan<- interface{}) error
}

type Runner interface {
	BeforeFunc(func(ctx context.Context) error) Runner
	AfterFunc(func(ctx context.Context, err error) error) Runner
	SetDeadline(t time.Time) Runner
	SetTimeout(duration time.Duration) Runner
	Send(in interface{})
	InFrom(w ...Runner) Runner
	Out(out interface{})
	SetOut(chan interface{})
	Start() Runner
	Wait() error
}

type runner struct {
	ctx        context.Context
	stopReqCtx context.Context
	cancel     context.CancelFunc
	stopReq     context.CancelFunc
	inChan     chan interface{}
	outChan    chan interface{}
	signalChan chan os.Signal
	limiter    chan struct{}

	afterFunc  func(ctx context.Context, err error) error
	workFunc   func(in interface{}, out chan<- interface{}) error
	beforeFunc func(ctx context.Context) error

	leader bool

	timeout  time.Duration
	deadline time.Duration

	err        error
	numWorkers int64
	lock       *sync.RWMutex
	wg         *sync.WaitGroup
	firstCall  *sync.Once
	once       *sync.Once
}

func NewRunner(ctx context.Context, w Worker, numWorkers int64) Runner {
	var runnerCtx, runnerCancel = context.WithCancel(ctx)
	stopReqCtx, stopReq := context.WithCancel(ctx)
	var runner = &runner{
		ctx:        runnerCtx,
		cancel:     runnerCancel,
		stopReqCtx: stopReqCtx,
		stopReq: stopReq,
		inChan:     make(chan interface{}, numWorkers),
		outChan:    nil,
		signalChan: make(chan os.Signal, 1),
		limiter:    make(chan struct{}, numWorkers),
		afterFunc:  func(ctx context.Context, err error) error { return err },
		workFunc:   w.Work,
		beforeFunc: func(ctx context.Context) error { return nil },
		leader:     true,
		err:        nil,
		numWorkers: numWorkers,
		lock:       new(sync.RWMutex),
		wg:         new(sync.WaitGroup),
		once:       new(sync.Once),
		firstCall:  new(sync.Once),
	}
	runner.waitForSignal(defaultWatchSignals...)
	return runner
}

func (r *runner) Send(in interface{}) {
	select {
	case <-r.ctx.Done():
		return
	case r.inChan <- in:
	}
}

func (r *runner) InFrom(w ...Runner) Runner {
	//r.inChan = make(chan interface{}, r.numWorkers*2)
	for _, wr := range w {
		wr.SetOut(r.inChan)
	}
	r.leader = false
	return r
}

func (r *runner) Start() Runner {
	r.startWork()
	return r
}

func (r *runner) BeforeFunc(f func(ctx context.Context) error) Runner {
	r.beforeFunc = f
	return r
}

func (r *runner) AfterFunc(f func(ctx context.Context, err error) error) Runner {
	r.afterFunc = f
	return r
}

func (r *runner) Out(out interface{}) {
	select {
	case <-r.ctx.Done():
		return
	case r.outChan <- out:
	}
}

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

func (r *runner) Wait() error {
	var err error
	r.stopReq()


	if err = <-r.errChan; err != nil && err != context.Canceled {
		return err
	}
	return r.afterFunc(r.ctx, err)
}

func (r *runner) waitImpl() error {
	r.wg.Wait()

}

func (r *runner) StopRequested() <-chan struct{} {
	return r.stopReqCtx.Done()
}

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

func (r *runner) startWork() {
	if r.err = r.beforeFunc(r.ctx); r.err != nil {
		r.errChan <- r.err
		return
	}
	if r.timeout > 0 {
		r.ctx, r.cancel = context.WithTimeout(r.ctx, r.timeout)
	}

	go func() {
		var wg = new(sync.WaitGroup)
		for {
			select {
			case <-r.StopRequested():
				wg.Wait()
				if len(r.inChan) > 0 {
					continue
				}
				r.cancel()
			case in := <-r.inChan:
				r.limiter <- struct{}{}
				wg.Add(1)
				go func() {
					defer func() {
						<-r.limiter
						wg.Done()
					}()
					if err := r.workFunc(in, r.outChan); err != nil {
						r.once.Do(func() {
							r.err = err
							r.cancel()
						})
					}
				}()
			case <-r.IsDone():
				if r.err == nil {
					r.err = context.Canceled
				}
				return
			}
		}
	}()
	return
}
