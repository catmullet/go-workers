package workers

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type WorkerObject interface {
	Work(w *Worker, in interface{}) error
}

// Worker The object to hold all necessary configuration and channels for the worker
// only accessible by it's methods.
type Worker struct {
	numberOfWorkers int
	Ctx             context.Context
	workerFunction  WorkerObject
	inChan          chan interface{}
	outChan         chan interface{}
	timeout         time.Duration
	cancel          context.CancelFunc
	sigChan         chan os.Signal
	err             error
	wg              sync.WaitGroup
	sync.Once
}

// NewWorker factory method to return new Worker
func NewWorker(ctx context.Context, workerFunction WorkerObject, numberOfWorkers int) (worker *Worker) {
	cctx, cancel := context.WithCancel(ctx)
	if numberOfWorkers < 1 {
		numberOfWorkers = 1
	}
	return &Worker{
		numberOfWorkers: numberOfWorkers,
		Ctx:             cctx,
		workerFunction:  workerFunction,
		inChan:          make(chan interface{}, numberOfWorkers),
		timeout:         time.Duration(0),
		cancel:          cancel,
		sigChan:         make(chan os.Signal, 1),
		wg:              sync.WaitGroup{},
		Once:            sync.Once{},
	}
}

// Send wrapper to send interface through workers "in" channel
func (iw *Worker) Send(in interface{}) {
	select {
	case <-iw.IsDone():
		return
	default:
		iw.inChan <- in
	}
}

// InFrom assigns workers out channel to this workers in channel
func (iw *Worker) InFrom(inWorker ...*Worker) *Worker {
	for _, worker := range inWorker {
		worker.outChan = iw.inChan
	}
	return iw
}

// Work start up the number of workers specified by the numberOfWorkers variable
func (iw *Worker) Work() *Worker {
	if iw.timeout > 0 {
		iw.Ctx, iw.cancel = context.WithTimeout(iw.Ctx, iw.timeout)
	}
	for i := 0; i < iw.numberOfWorkers; i++ {
		iw.wg.Add(1)
		go func(iw *Worker) {
			defer iw.wg.Done()
			if err := iw.workFunc(); err != nil {
				iw.Do(func() {
					iw.err = err
					if iw.cancel != nil {
						iw.cancel()
					}
				})
			}
		}(iw)
	}
	return iw
}

func (iw *Worker) workFunc() error {
	for {
		select {
		case <-iw.IsDone():
			return nil
		case in := <-iw.inChan:
			if err := iw.workerFunction.Work(iw, in); err != nil {
				return err
			}
		}
	}
}

// In returns the workers in channel
func (iw *Worker) In() chan interface{} {
	return iw.inChan
}

// Out pushes value to workers out channel
func (iw *Worker) Out(out interface{}) {
	select {
	case <-iw.Ctx.Done():
		return
	default:
		iw.outChan <- out
	}
}

// wait waits for all the workers to finish up
func (iw *Worker) wait() (err error) {
	iw.wg.Wait()
	if iw.cancel != nil {
		iw.cancel()
	}
	return iw.err
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
