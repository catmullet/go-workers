package workers

import (
	"context"
	"errors"
	"golang.org/x/sync/errgroup"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type WorkerObject interface {
	Work(w *Worker, in interface{}) error
}

// Worker The object to hold all necessary configuration and channels for the worker
// only accessible by it's methods.
type Worker struct {
	numberOfWorkers               int64
	Ctx                           context.Context
	workerFunction                WorkerObject
	inChan                        chan interface{}
	outChan                       chan interface{}
	inScalingChan                 chan int64
	outScalingChan                chan int64
	lock                          *sync.RWMutex
	timeout                       time.Duration
	cancel                        context.CancelFunc
	sigChan                       chan os.Signal
	errGroup                      *errgroup.Group
	autoScaleWorkersMax           int64
	availableAutoScaleWorkers     int64
	autoScaleCooldownMilliseconds int64
	cooldownChan                  chan int64
}

// NewWorker factory method to return new Worker
func NewWorker(ctx context.Context, workerFunction WorkerObject, numberOfWorkers int64) (worker *Worker) {
	cctx, cancel := context.WithCancel(ctx)
	worker = &Worker{
		numberOfWorkers:               numberOfWorkers,
		Ctx:                           cctx,
		workerFunction:                workerFunction,
		inChan:                        make(chan interface{}, numberOfWorkers),
		inScalingChan:                 make(chan int64),
		outScalingChan:                make(chan int64),
		cooldownChan:                  make(chan int64),
		lock:                          new(sync.RWMutex),
		timeout:                       time.Duration(0),
		cancel:                        cancel,
		sigChan:                       make(chan os.Signal, 1),
		autoScaleWorkersMax:           0,
		availableAutoScaleWorkers:     0,
		autoScaleCooldownMilliseconds: 300,
	}
	worker.errGroup, _ = errgroup.WithContext(ctx)
	return
}

// Send wrapper to send interface through workers "in" channel
func (iw *Worker) Send(in interface{}) {
	select {
	case <-iw.IsDone():
		return
	case iw.inChan <- in:
	default:
		iw.inScalingChan <- 1
		iw.Send(in)
	}
}

// InFrom assigns workers out channel to this workers in channel
func (iw *Worker) InFrom(inWorker ...*Worker) *Worker {
	for _, worker := range inWorker {
		worker.outScalingChan = iw.inScalingChan
		worker.outChan = iw.inChan
	}
	return iw
}

// Work start up the number of workers specified by the numberOfWorkers variable
func (iw *Worker) Work() *Worker {
	if iw.timeout > 0 {
		iw.Ctx, iw.cancel = context.WithTimeout(iw.Ctx, iw.timeout)
	}
	for i := int64(0); i < atomic.LoadInt64(&iw.numberOfWorkers); i++ {
		iw.addWorker()
	}
	iw.autoscaler()
	return iw
}

func (iw *Worker) autoscaler() {
	iw.errGroup.Go(func() error {
		for {
			select {
			case <-iw.IsDone():
				return context.Canceled
			case <-iw.inScalingChan:
				if atomic.LoadInt64(&iw.availableAutoScaleWorkers) > 0 {
					iw.addBufferedWorker()
					atomic.AddInt64(&iw.numberOfWorkers, 1)
					atomic.AddInt64(&iw.availableAutoScaleWorkers, -1)
				}
			}
		}
	})
}

func (iw *Worker) WorkerCount() int64 {
	return atomic.LoadInt64(&iw.numberOfWorkers)
}

// AddBuffer allows slower workers to increase their ability to catch up to faster workers
func (iw *Worker) AddBuffer(count int64) *Worker {
	if count > 0 {
		atomic.AddInt64(&iw.autoScaleWorkersMax, count)
		atomic.AddInt64(&iw.availableAutoScaleWorkers, count)
	}
	return iw
}

func (iw *Worker) SetBufferTimeoutMS(ms int64) *Worker {
	atomic.StoreInt64(&iw.autoScaleCooldownMilliseconds, ms)
	return iw
}

// addWorker add a worker to the pool.
func (iw *Worker) addWorker() {
	iw.errGroup.Go(func() error {
		for {
			select {
			case <-iw.IsDone():
				atomic.AddInt64(&iw.numberOfWorkers, -1)
				return context.Canceled
			case in := <-iw.In():
				err := iw.workerFunction.Work(iw, in)
				if err != nil {
					return err
				}
			}
		}
	})
}

// addWorker add a worker to the pool.
func (iw *Worker) addBufferedWorker() {
	iw.errGroup.Go(func() error {
		tm := time.NewTicker(iw.getCooldownMSDuration())
		for {
			select {
			case <-iw.IsDone():
				atomic.AddInt64(&iw.numberOfWorkers, -1)
				return context.Canceled
			case in := <-iw.In():
				err := iw.workerFunction.Work(iw, in)
				if err != nil {
					return err
				}
				tm.Reset(iw.getCooldownMSDuration())
			case <-tm.C:
				atomic.AddInt64(&iw.numberOfWorkers, -1)
				atomic.AddInt64(&iw.availableAutoScaleWorkers, 1)
				return nil
			}
		}
	})
}

func (iw *Worker) getCooldownMSDuration() time.Duration {
	return time.Duration(atomic.LoadInt64(&iw.autoScaleCooldownMilliseconds)) * time.Millisecond
}

// In returns the workers in channel
func (iw *Worker) In() chan interface{} {
	return iw.inChan
}

// Out pushes value to workers out channel
func (iw *Worker) Out(out interface{}) {
	select {
	case iw.outChan <- out:
	default:
		iw.outScalingChan <- 1
		iw.Out(out)
	}
}

// wait waits for all the workers to finish up
func (iw *Worker) wait() (err error) {
	err = iw.errGroup.Wait()
	return
}

// Cancel stops all workers
func (iw *Worker) Cancel() {
	iw.cancel()
}

// SetDeadline allows a time to be set when the workers should stop.
// Deadline needs to be handled by the IsDone method.
func (iw *Worker) SetDeadline(t time.Time) *Worker {
	iw.Ctx, iw.cancel = context.WithDeadline(iw.Ctx, t)
	return iw
}

// SetTimeout allows a time duration to be set when the workers should stop.
// Timeout needs to be handled by the IsDone method.
func (iw *Worker) SetTimeout(duration time.Duration) *Worker {
	iw.timeout = duration
	return iw
}

// IsDone returns a context's cancellation or error
func (iw *Worker) IsDone() <-chan struct{} {
	return iw.Ctx.Done()
}

// Close Note that it is only necessary to close a channel if the receiver is
// looking for a close. Closing the channel is a control signal on the
// channel indicating that no more data follows. Thus it makes sense to only close the
// in channel on the worker. For now we will just send the cancel signal
func (iw *Worker) Close() error {
	defer close(iw.inChan)
	iw.Cancel()
	if err := iw.wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// PassTerminationSignal passes signal to workers
func (iw *Worker) PassTerminationSignal(sig os.Signal) {
	iw.sigChan <- sig
}

// SafeShutdown wait for an interrupt or termination signal and
// if encountered handle clean shut down/cleanup.
func (iw *Worker) SafeShutdown() *Worker {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func(chan os.Signal) {
		sig := <-sigChan
		log.Printf("workers recieved termination signal: %s", sig.String())
		if err := iw.Close(); err != nil {
			log.Printf("workers safely shutdown but recieved error: %s", err.Error())
		}
	}(sigChan)
	return iw
}
