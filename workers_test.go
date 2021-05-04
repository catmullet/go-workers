package workers

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

const (
	workerCount   = 1000
	workerTimeout = time.Millisecond * 300
	runTimes      = 100000
)

type WorkerOne struct {
}
type WorkerTwo struct {
}

func NewWorkerOne() Worker {
	return &WorkerOne{}
}

func NewWorkerTwo() Worker {
	return &WorkerTwo{}
}

func (wo *WorkerOne) Work(in interface{}, out chan<- interface{}) error {
	var workerOne = "worker_one"
	mut.Lock()
	if val, ok := count[workerOne]; ok {
		count[workerOne] = val + 1
	} else {
		count[workerOne] = 1
	}
	mut.Unlock()

	total := in.(int) * 2
	out <- total
	return nil
}

func (wt *WorkerTwo) Work(in interface{}, out chan<- interface{}) error {
	var workerTwo = "worker_two"
	mut.Lock()
	if val, ok := count[workerTwo]; ok {
		count[workerTwo] = val + 1
	} else {
		count[workerTwo] = 1
	}
	mut.Unlock()
	return nil
}

var (
	count               = make(map[string]int)
	mut                 = sync.RWMutex{}
	err                 = errors.New("test error")
	deadline            = func() time.Time { return time.Now().Add(workerTimeout) }
	workerTestScenarios = []workerTest{
		{
			name:       "work basic",
			worker:     NewTestWorkerObject(workBasic()),
			numWorkers: workerCount,
		},
		{
			name:       "work basic with timeout",
			timeout:    workerTimeout,
			worker:     NewTestWorkerObject(workBasic()),
			numWorkers: workerCount,
		},
		{
			name:       "work basic with deadline",
			deadline:   deadline,
			worker:     NewTestWorkerObject(workBasic()),
			numWorkers: workerCount,
		},
		{
			name:        "work with return of error",
			worker:      NewTestWorkerObject(workWithError(err)),
			errExpected: true,
			numWorkers:  workerCount,
		},
		{
			name:        "work with return of error with timeout",
			timeout:     workerTimeout,
			worker:      NewTestWorkerObject(workWithError(err)),
			errExpected: true,
			numWorkers:  workerCount,
		},
		{
			name:        "work with return of error with deadline",
			deadline:    deadline,
			worker:      NewTestWorkerObject(workWithError(err)),
			errExpected: true,
			numWorkers:  workerCount,
		},
	}

	getWorker = func(ctx context.Context, wt workerTest) Runner {
		worker := NewRunner(ctx, wt.worker, wt.numWorkers)
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
	name        string
	timeout     time.Duration
	deadline    func() time.Time
	worker      Worker
	numWorkers  int64
	testSignal  bool
	errExpected bool
}

type TestWorkerObject struct {
	workFunc func(in interface{}, out chan<- interface{}) error
}

func NewTestWorkerObject(wf func(in interface{}, out chan<- interface{}) error) Worker {
	return &TestWorkerObject{wf}
}

func (tw *TestWorkerObject) Work(in interface{}, out chan<- interface{}) error {
	return tw.workFunc(in, out)
}

func workBasicNoOut() func(in interface{}, out chan<- interface{}) error {
	return func(in interface{}, out chan<- interface{}) error {
		_ = in.(int)
		return nil
	}
}

func workBasic() func(in interface{}, out chan<- interface{}) error {
	return func(in interface{}, out chan<- interface{}) error {
		i := in.(int)
		out <- i
		return nil
	}
}

func workWithError(err error) func(in interface{}, out chan<- interface{}) error {
	return func(in interface{}, out chan<- interface{}) error {
		i := in.(int)
		total := i * rand.Intn(1000)
		if i == 100 {
			return err
		}
		out <- total
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
			workerOne := getWorker(ctx, tt).Start()
			// always need a consumer for the out tests so using basic here.
			workerTwo := NewRunner(ctx, NewTestWorkerObject(workBasicNoOut()), workerCount).InFrom(workerOne).Start()

			for i := 0; i < runTimes; i++ {
				workerOne.Send(i)
			}

			if err := workerOne.Wait(); err != nil && !tt.errExpected {
				fmt.Println(err)
				t.Fail()
			}
			if err := workerTwo.Wait(); err != nil && !tt.errExpected {
				fmt.Println(err)
				t.Fail()
			}
		})
	}
}

func TestWorkersFinish(t *testing.T) {
	ctx := context.Background()
	workerOne := NewRunner(ctx, NewWorkerOne(), 1000).Start()
	workerTwo := NewRunner(ctx, NewWorkerTwo(), 1000).InFrom(workerOne).Start()

	for i := 0; i < 100000; i++ {
		workerOne.Send(rand.Intn(100))
	}

	if err := workerOne.Wait(); err != nil {
		fmt.Println(err)
	}

	if err := workerTwo.Wait(); err != nil {
		fmt.Println(err)
	}

	if count["worker_one"] != 100000 {
		fmt.Println("worker one failed to finish,", "worker_one count", count["worker_one"], "/ 100000")
		t.Fail()
	}
	if count["worker_two"] != 100000 {
		fmt.Println("worker two failed to finish,", "worker_two count", count["worker_two"], "/ 100000")
		t.Fail()
	}
}

func BenchmarkGoWorkers(b *testing.B) {
	ctx := context.Background()
	worker := NewRunner(ctx, NewTestWorkerObject(workBasicNoOut()), workerCount).Start()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < runTimes; j++ {
			worker.Send(j)
		}
	}

	b.StopTimer()
	if err := worker.Wait(); err != nil {
		b.Error(err)
	}
}
