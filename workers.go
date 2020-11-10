package goworker

import (
	"context"
	"reflect"
	"sync"
	"time"
)

type fields map[interface{}]interface{}

// Worker The object to hold all necessary configuration and channels for the worker
// only accessible by it's methods.
type Worker struct {
	numberOfWorkers int
	Ctx             context.Context
	workerFunction  func(ig *Worker) (err error)
	inChan          chan interface{}
	outChan         chan interface{}
	lock            *sync.RWMutex
	timeout         time.Duration
	cancel		context.CancelFunc
	fields          fields
	errGroup        *errGroup
}

// NewWorker factory method to return new Worker
func NewWorker(ctx context.Context, workerFunction func(ig *Worker) (err error), numberOfWorkers int) (worker *Worker) {
	worker = &Worker{
		numberOfWorkers: numberOfWorkers,
		Ctx:             ctx,
		workerFunction:  workerFunction,
		inChan:          make(chan interface{}),
		outChan:         make(chan interface{}),
		timeout:         time.Duration(0),
		lock:            new(sync.RWMutex),
		fields:          make(fields),
		errGroup:        nil,
	}

	worker.errGroup, _ = errGroupWithContext(ctx)
	return
}

// Send wrapper to send interface through workers "in" channel
func (iw *Worker) Send(in interface{}) {
	if iw.errGroup.err != nil {
		return
	}
	iw.inChan <- in
}

// InFrom assigns workers out channel to this workers in channel
func (iw *Worker) InFrom(inWorker ...*Worker) *Worker {
	for _, worker := range inWorker {
		worker.outChan = iw.inChan
	}
	return iw
}

// AddField Adds a variable, struct or pointer by key value
func (iw *Worker) AddField(key interface{}, value interface{}) *Worker {
	iw.lock.Lock()
	defer iw.lock.Unlock()
	iw.fields[key] = value
	return iw
}

// Work start up the number of workers specified by the numberOfWorkers variable
func (iw *Worker) Work() *Worker {
	if iw.timeout > 0 {
		iw.Ctx, iw.cancel = context.WithTimeout(iw.Ctx, iw.timeout)
	}
	for i := 0; i < iw.numberOfWorkers; i++ {
		iw.errGroup.goWork(iw.workerFunction, iw)
	}
	return iw
}

// BindField returns value of key as passed in objects type
func (iw *Worker) BindField(name string, obj interface{}) (ok bool) {
	iw.lock.RLock()
	defer iw.lock.RUnlock()
	if param, ok := iw.fields[name]; ok {

		iv := reflect.ValueOf(param)
		ov := reflect.ValueOf(obj)

		if ov.Kind() == reflect.Ptr {
			if iv.Type() == ov.Elem().Type() {
				ov.Elem().Set(iv)
			} else {
				ov.Elem().Set(iv.Elem())
			}
		}
	}
	return
}

// In returns the workers in channel
func (iw *Worker) In() chan interface{} {
	return iw.inChan
}

// Out pushes value to workers out channel
func (iw *Worker) Out(out interface{}) {
	iw.outChan <- out
}

// Wait wait for all the workers to finish up
func (iw *Worker) Wait() (err error) {
	err = iw.errGroup.wait()
	return
}

// Cancel stops all workers
func (iw *Worker) Cancel() {
	iw.cancel()
	iw.errGroup.cancel()
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
// in channel on the worker.
func (iw *Worker) Close() {
	close(iw.inChan)
}
