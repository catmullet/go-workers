package goworker

import (
	"context"
	"reflect"
	"sync"
)

type Fields map[interface{}]interface{}

type Worker struct {
	numberOfWorkers int
	Ctx             context.Context
	workerFunction  func(ig *Worker) (err error)
	inChan          chan interface{}
	outChan         chan interface{}
	lock            *sync.RWMutex
	timerChan       chan bool
	fields          Fields
	errGroup        *ErrGroup
}

// NewWorker factory method to return new Worker
func NewWorker(ctx context.Context, workerFunction func(ig *Worker) (err error), numberOfWorkers int) (worker *Worker) {
	worker = &Worker{
		numberOfWorkers: numberOfWorkers,
		Ctx:             ctx,
		workerFunction:  workerFunction,
		inChan:          make(chan interface{}),
		outChan:         make(chan interface{}),
		timerChan:       make(chan bool),
		lock:            new(sync.RWMutex),
		fields:          make(Fields),
		errGroup:        nil,
	}

	worker.errGroup, _ = ErrGroupWithContext(ctx)
	return
}

// Send wrapper to send interface through workers "in" channel
func (iw *Worker) Send(in interface{}) {
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
	for i := 0; i < iw.numberOfWorkers; i++ {
		iw.errGroup.Go(iw.workerFunction, iw)
	}
	return iw
}

// GetFieldString returns value of key as string
func (iw *Worker) GetFieldString(name string) (st string, ok bool) {
	iw.lock.RLock()
	defer iw.lock.RUnlock()
	if param, exists := iw.fields[name]; exists {
		if st, ok = param.(string); ok {
			return
		}
	}
	return
}

// GetFieldInt returns value of key as int
func (iw *Worker) GetFieldInt(name string) (in int, ok bool) {
	iw.lock.RLock()
	defer iw.lock.RUnlock()
	if param, exists := iw.fields[name]; exists {
		if in, ok = param.(int); ok {
			return
		}
	}
	return
}

// GetFieldInt32 returns value of key as int32
func (iw *Worker) GetFieldInt32(name string) (in int32, ok bool) {
	iw.lock.RLock()
	defer iw.lock.RUnlock()
	if param, exists := iw.fields[name]; exists {
		if in, ok = param.(int32); ok {
			return
		}
	}
	return
}

// GetFieldInt64 returns value of key as int64
func (iw *Worker) GetFieldInt64(name string) (in int64, ok bool) {
	iw.lock.RLock()
	defer iw.lock.RUnlock()
	if param, exists := iw.fields[name]; exists {
		if in, ok = param.(int64); ok {
			return
		}
	}
	return
}

// GetFieldFloat32 returns value of key as float32
func (iw *Worker) GetFieldFloat32(name string) (fl float32, ok bool) {
	iw.lock.RLock()
	defer iw.lock.RUnlock()
	if param, exists := iw.fields[name]; exists {
		if fl, ok = param.(float32); ok {
			return
		}
	}
	return
}

// GetFieldFloat64 returns value of key as float64
func (iw *Worker) GetFieldFloat64(name string) (fl float64, ok bool) {
	iw.lock.RLock()
	defer iw.lock.RUnlock()
	if param, exists := iw.fields[name]; exists {
		if fl, ok = param.(float64); ok {
			return
		}
	}
	return
}

// GetFieldBool returns value of key as bool
func (iw *Worker) GetFieldBool(name string) (bl bool, ok bool) {
	iw.lock.RLock()
	defer iw.lock.RUnlock()
	if param, exists := iw.fields[name]; exists {
		if bl, ok = param.(bool); ok {
			return
		}
	}
	return
}

// GetFieldObject returns value of key as passed in objects type
func (iw *Worker) GetFieldObject(name string, obj interface{}) (ok bool) {
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
	err = iw.errGroup.Wait()
	return
}

// Cancel stops all workers
func (iw *Worker) Cancel() {
	iw.errGroup.cancel()
}

// Close Note that it is only necessary to close a channel if the receiver is
// looking for a close. Closing the channel is a control signal on the
// channel indicating that no more data follows. Thus it makes sense to only close the
// in channel on the worker.
func (iw *Worker) Close() {
	close(iw.inChan)
}
