package workers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"time"
)

const (
	internalBufferFlushLimit = 512
	minNumberOfWorkersLimit  = 1
	signalChannelBufferSize  = 1
)

// WorkerObject interface to be implemented
type WorkerObject interface {
	Work(w *Worker, in interface{}) error
}

// Worker The object to hold all necessary configuration and channels for the worker
// only accessible by it's methods.
type Worker struct {
	Ctx             context.Context
	workerFunction  WorkerObject
	err             error
	numberOfWorkers int
	inChan          chan interface{}
	outChan         chan interface{}
	sigChan         chan os.Signal
	timeout         time.Duration
	cancel          context.CancelFunc
	writer          *bufio.Writer
	mu              *sync.RWMutex
	wg              *sync.WaitGroup
	once            *sync.Once
}

// NewWorker factory method to return new Worker
func NewWorker(ctx context.Context, workerFunction WorkerObject, numberOfWorkers int) (worker *Worker) {
	c, cancel := context.WithCancel(ctx)
	numberOfWorkers = getCorrectWorkers(numberOfWorkers)
	return &Worker{
		numberOfWorkers: numberOfWorkers,
		Ctx:             c,
		workerFunction:  workerFunction,
		inChan:          make(chan interface{}, numberOfWorkers),
		sigChan:         make(chan os.Signal, signalChannelBufferSize),
		timeout:         time.Duration(0),
		cancel:          cancel,
		writer:          bufio.NewWriter(os.Stdout),
		wg:              new(sync.WaitGroup),
		mu:              new(sync.RWMutex),
		once:            new(sync.Once),
	}
}

// Send wrapper to send interface through workers "in" channel
func (iw *Worker) Send(in interface{}) {
	select {
	case <-iw.IsDone():
		return
	case iw.inChan <- in:
		return
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
	go func() {
		iw.wg.Add(1)
		defer iw.wg.Done()
		for {
			select {
			case <-iw.IsDone():
				if len(iw.inChan) > 0 {
					continue
				}
				if iw.err == nil {
					iw.err = context.Canceled
				}
				return
			case in := <-iw.inChan:
				iw.wg.Add(1)
				go func(in interface{}) {
					defer iw.wg.Done()
					if err := iw.workerFunction.Work(iw, in); err != nil {
						iw.once.Do(func() {
							iw.err = err
							if iw.cancel != nil {
								iw.cancel()
							}
						})
					}
				}(in)
			}
		}
	}()
	return iw
}

// Out pushes value to workers out channel
func (iw *Worker) Out(out interface{}) {
	select {
	case <-iw.Ctx.Done():
		return
	case iw.outChan <- out:
		return
	}
}

// CancelOnSignal will send cancel to workers when signals specified are received
func (iw *Worker) CancelOnSignal(signals ...os.Signal) *Worker {
	if len(signals) > 0 {
		iw.waitForSignal(signals...)
	}
	return iw
}

// waitForSignal make sure we wait for a term signal and shutdown correctly
func (iw *Worker) waitForSignal(signals ...os.Signal) {
	go func() {
		signal.Notify(iw.sigChan, signals...)
		<-iw.sigChan
		if iw.cancel != nil {
			iw.cancel()
		}
	}()
}

// getCorrectWorkers don't let oversizing occur on workers
func getCorrectWorkers(numberOfWorkers int) int {
	if numberOfWorkers < minNumberOfWorkersLimit {
		numberOfWorkers = minNumberOfWorkersLimit
	}

	return numberOfWorkers
}

// Wait waits for all the workers to finish up
func (iw *Worker) Wait() (err error) {
	iw.wg.Wait()
	if iw.cancel != nil {
		iw.cancel()
	}
	return iw.err
}

// SetDeadline allows a time to be set when the workers should stop.
// Deadline needs to be handled by the IsDone method.
func (iw *Worker) SetDeadline(t time.Time) *Worker {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.Ctx, iw.cancel = context.WithDeadline(iw.Ctx, t)
	return iw
}

// SetTimeout allows a time duration to be set when the workers should stop.
// Timeout needs to be handled by the IsDone method.
func (iw *Worker) SetTimeout(duration time.Duration) *Worker {
	iw.mu.Lock()
	defer iw.mu.Unlock()
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
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.internalBufferFlush()
	_, _ = iw.writer.WriteString(fmt.Sprintln(a...))
}

// Printf prints based on format provided and a
func (iw *Worker) Printf(format string, a ...interface{}) {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.internalBufferFlush()
	_, _ = iw.writer.WriteString(fmt.Sprintf(format, a...))
}

// Print prints output of a
func (iw *Worker) Print(a ...interface{}) {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.internalBufferFlush()
	_, _ = iw.writer.WriteString(fmt.Sprint(a...))
}

// internalBufferFlush makes sure we haven't used up the available buffer
// by flushing the buffer when we get below the danger zone.
func (iw *Worker) internalBufferFlush() {
	if iw.writer.Available() < internalBufferFlushLimit {
		_ = iw.writer.Flush()
	}
}

// Close Note that it is only necessary to close a channel if the receiver is
// looking for a close. Closing the channel is a control signal on the
// channel indicating that no more data follows. Thus it makes sense to only close the
// in channel on the worker. For now we will just send the cancel signal
func (iw *Worker) Close() error {
	iw.cancel()
	defer func() { _ = iw.writer.Flush() }()
	if err := iw.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
