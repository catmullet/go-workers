package workers

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

const (
	workerCount   = 1000
	workerTimeout = time.Millisecond * 300
)

var (
	err                 = errors.New("test error")
	deadline            = func() time.Time { return time.Now().Add(workerTimeout) }
	workerTestScenarios = []workerTest{
		{
			name:         "work basic",
			workerObject: NewTestWorkerObject(workBasic()),
			numWorkers:   workerCount,
		},
		{
			name:         "work basic with timeout",
			timeout:      workerTimeout,
			workerObject: NewTestWorkerObject(workBasic()),
			numWorkers:   workerCount,
		},
		{
			name:         "work basic with deadline",
			deadline:     deadline,
			workerObject: NewTestWorkerObject(workBasic()),
			numWorkers:   workerCount,
		},
		{
			name:         "work with return of error",
			workerObject: NewTestWorkerObject(workWithError(err)),
			errExpected:  true,
			numWorkers:   workerCount,
		},
		{
			name:         "work with return of error with timeout",
			timeout:      workerTimeout,
			workerObject: NewTestWorkerObject(workWithError(err)),
			errExpected:  true,
			numWorkers:   workerCount,
		},
		{
			name:         "work with return of error with deadline",
			deadline:     deadline,
			workerObject: NewTestWorkerObject(workWithError(err)),
			errExpected:  true,
			numWorkers:   workerCount,
		},
	}

	getWorker = func(ctx context.Context, wt workerTest) *Worker {
		worker := NewWorker(ctx, wt.workerObject, wt.numWorkers)
		if wt.timeout > 0 {
			worker.SetTimeout(wt.timeout)
		}
		if wt.deadline != nil {
			worker.SetDeadline(wt.deadline())
		}
		return worker
	}
)

type workerTest struct {
	name         string
	timeout      time.Duration
	deadline     func() time.Time
	workerObject WorkerObject
	numWorkers   int
	errExpected  bool
}

type TestWorkerObject struct {
	workFunc func(w *Worker, in interface{}) error
}

func NewTestWorkerObject(wf func(w *Worker, in interface{}) error) *TestWorkerObject {
	return &TestWorkerObject{wf}
}

func (tw *TestWorkerObject) Work(w *Worker, in interface{}) error {
	return tw.workFunc(w, in)
}

func workBasicNoOut() func(w *Worker, in interface{}) error {
	return func(w *Worker, in interface{}) error {
		_ = in.(int)
		return nil
	}
}

func workBasic() func(w *Worker, in interface{}) error {
	return func(w *Worker, in interface{}) error {
		i := in.(int)
		total := i * rand.Intn(1000)
		w.Out(total)
		return nil
	}
}

func workWithError(err error) func(w *Worker, in interface{}) error {
	return func(w *Worker, in interface{}) error {
		i := in.(int)
		total := i * rand.Intn(1000)
		if i == 1000 {
			return err
		}
		w.Out(total)
		return nil
	}
}

func TestMain(m *testing.M) {
	debug.SetGCPercent(500)
	runtime.GOMAXPROCS(2)
	code := m.Run()
	os.Exit(code)
}

func TestWorkers(t *testing.T) {
	for _, tt := range workerTestScenarios {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			workerOne := getWorker(ctx, tt).Work()
			// always need a consumer for the out tests so using basic here.
			workerTwo := NewWorker(ctx, NewTestWorkerObject(workBasicNoOut()), workerCount)
			workerTwo.InFrom(workerOne).Work()

			for i := 0; i < 10000000; i++ {
				workerOne.Send(i)
			}

			fmt.Println(tt.name, " waiting on worker one to close...")
			if err := workerOne.Close(); !assert.Equal(t, tt.errExpected, err != nil) {
				fmt.Println(err)
				t.Fail()
			}

			fmt.Println(tt.name, " waiting on worker two to close...")
			if err := workerTwo.Close(); !assert.NoError(t, err) {
				fmt.Println(err)
				t.Fail()
			}
		})
	}
}
