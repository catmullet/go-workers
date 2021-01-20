package workers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// WorkerObject interface to be implemented
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
	err             error
	wg              sync.WaitGroup
	writer          *bufio.Writer
	sync.RWMutex
	sync.Once
}

// NewWorker factory method to return new Worker
func NewWorker(ctx context.Context, workerFunction WorkerObject, numberOfWorkers int) (worker *Worker) {
	debug.SetGCPercent(500)
	c, cancel := context.WithCancel(ctx)
	numberOfWorkers = getCorrectWorkers(numberOfWorkers)
	return &Worker{
		numberOfWorkers: numberOfWorkers,
		Ctx:             c,
		workerFunction:  workerFunction,
		inChan:          make(chan interface{}, numberOfWorkers),
		timeout:         time.Duration(0),
		cancel:          cancel,
		writer:          bufio.NewWriter(os.Stdout),
		wg:              sync.WaitGroup{},
		RWMutex:         sync.RWMutex{},
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
		upMaxProcs()
		worker.outChan = iw.inChan
	}
	return iw
}

// Work start up the number of workers specified by the numberOfWorkers variable
func (iw *Worker) Work() *Worker {
	setDefaultMaxProcs()
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

// workFunc separated work func from concurrency mess.
func (iw *Worker) workFunc() error {
	for {
		select {
		case <-iw.IsDone():
			return nil
		case in := <-iw.inChan:
			if err := iw.workerFunction.Work(iw, in); err != nil &&
				err != context.Canceled {
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

// getCorrectWorkers don't let oversizing occur on workers
func getCorrectWorkers(numberOfWorkers int) int {
	if numberOfWorkers < 1 {
		numberOfWorkers = 1
	}
	if numberOfWorkers > 1000 {
		numberOfWorkers = 1000
	}
	return numberOfWorkers
}

// DisableAutoMaxProcs sets MaxProcs to default.
func (iw *Worker) DisableAutoMaxProcs() *Worker {
	runtime.GOMAXPROCS(runtime.NumCPU())
	return iw
}

// setDefaultMaxProcs sets the max procs to a default of 1.
// we are assuming this is the only concurrency the application is
// going to be using. Why wouldn't you use all or nothing, right?
// If our assumption was wrong there is always DisableAutoMaxProcs().
// Hope you know what your doing!
func setDefaultMaxProcs() {
	runtime.GOMAXPROCS(1)
}

// upMaxProcs up the maxProcs by 1. Won't pass max numCPU.
func upMaxProcs() {
	prev := runtime.GOMAXPROCS(2)
	if prev >= 2 {
		if prev+1 <= runtime.NumCPU() {
			runtime.GOMAXPROCS(prev + 1)
		}
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

// SetWriterOut sets the writer for the Print* functions
// (ex.
// 		f, err := os.Create("output.txt"))
//		defer f.Close()
// 		worker.SetWriteOut(f)
// )
// If you have to print anything to stdout using the provided
// Print functions can significantly improve performance by using
// buffered output.
func (iw *Worker) SetWriterOut(writer io.Writer) *Worker {
	iw.writer.Reset(writer)
	return iw
}

// Println prints line output of a
func (iw *Worker) Println(a ...interface{}) {
	iw.Lock()
	defer iw.Unlock()
	iw.internalBufferFlush()
	_, _ = iw.writer.WriteString(fmt.Sprintln(a...))
}

// Printf prints based on format provided and a
func (iw *Worker) Printf(format string, a ...interface{}) {
	iw.Lock()
	defer iw.Unlock()
	iw.internalBufferFlush()
	_, _ = iw.writer.WriteString(fmt.Sprintf(format, a...))
}

// Print prints output of a
func (iw *Worker) Print(a ...interface{}) {
	iw.Lock()
	defer iw.Unlock()
	iw.internalBufferFlush()
	_, _ = iw.writer.WriteString(fmt.Sprint(a...))
}

// internalBufferFlush makes sure we haven't used up the available buffer
// by flushing the buffer when we get below the danger zone.
func (iw *Worker) internalBufferFlush() {
	if iw.writer.Available() < 512 {
		_ = iw.writer.Flush()
	}
}

// Close Note that it is only necessary to close a channel if the receiver is
// looking for a close. Closing the channel is a control signal on the
// channel indicating that no more data follows. Thus it makes sense to only close the
// in channel on the worker. For now we will just send the cancel signal
func (iw *Worker) Close() error {
	iw.Cancel()
	if err := iw.wait(); err != nil && !errors.Is(err, context.Canceled) {
		_ = iw.writer.Flush()
		return err
	}
	_ = iw.writer.Flush()
	return nil
}
