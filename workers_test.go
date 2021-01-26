package workers

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"path/filepath"
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
			name:         "work basic with Println",
			workerObject: NewTestWorkerObject(workBasicPrintln()),
			numWorkers:   workerCount,
		},
		{
			name:         "work basic with Printf",
			workerObject: NewTestWorkerObject(workBasicPrintf()),
			numWorkers:   workerCount,
		},
		{
			name:         "work basic with Print",
			workerObject: NewTestWorkerObject(workBasicPrint()),
			numWorkers:   workerCount,
		},
		{
			name:         "work basic less than minimum worker count",
			workerObject: NewTestWorkerObject(workBasic()),
			numWorkers:   0,
		},
		{
			name:         "work basic more than maximum worker count",
			workerObject: NewTestWorkerObject(workBasic()),
			numWorkers:   20000,
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
	testSignal   bool
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

func workBasicPrintln() func(w *Worker, in interface{}) error {
	return func(w *Worker, in interface{}) error {
		i := in.(int)
		w.Println(i)
		return nil
	}
}

func workBasicPrintf() func(w *Worker, in interface{}) error {
	return func(w *Worker, in interface{}) error {
		i := in.(int)
		w.Printf("test_number:%d", i)
		return nil
	}
}

func workBasicPrint() func(w *Worker, in interface{}) error {
	return func(w *Worker, in interface{}) error {
		i := in.(int)
		w.Print(i)
		return nil
	}
}

func workBasic() func(w *Worker, in interface{}) error {
	return func(w *Worker, in interface{}) error {
		i := in.(int)
		w.Out(i)
		return nil
	}
}

func workWithError(err error) func(w *Worker, in interface{}) error {
	return func(w *Worker, in interface{}) error {
		i := in.(int)
		total := i * rand.Intn(1000)
		if i == 100 {
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
	f, err := os.Create(filepath.Join(os.TempDir(), "testfile.txt"))
	if err != nil {
		t.Fail()
	}
	for _, tt := range workerTestScenarios {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			workerOne := getWorker(ctx, tt).SetWriterOut(f).Work()
			// always need a consumer for the out tests so using basic here.
			workerTwo := NewWorker(ctx, NewTestWorkerObject(workBasicNoOut()), workerCount)
			workerTwo.InFrom(workerOne).Work()

			for i := 0; i < 100000; i++ {
				workerOne.Send(i)
			}

			if err := workerOne.Close(); !assert.Equal(t, tt.errExpected, err != nil) {
				fmt.Println(err)
				t.Fail()
			}
			if err := workerTwo.Close(); !assert.NoError(t, err) {
				fmt.Println(err)
				t.Fail()
			}
		})
	}
}
