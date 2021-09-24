package gorkers_test

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guilhem/gorkers"
)

const (
	workerCount   = 100000
	workerTimeout = time.Millisecond * 300
	runTimes      = 100000
)

type WorkerOne struct {
	Count int32
	sync.Mutex
}
type WorkerTwo struct {
	Count int32
	sync.Mutex
}

func NewWorkerOne() *WorkerOne {
	return &WorkerOne{}
}

func NewWorkerTwo() *WorkerTwo {
	return &WorkerTwo{}
}

func (wo *WorkerOne) CurrentCount() int {
	return int(wo.Count)
}

func (wo *WorkerOne) Work(ctx context.Context, in interface{}, out chan<- interface{}) error {
	atomic.AddInt32(&wo.Count, 1)

	total := in.(int) * 2
	out <- total
	return nil
}

func (wt *WorkerTwo) CurrentCount() int {
	return int(wt.Count)
}

func (wt *WorkerTwo) Work(ctx context.Context, in interface{}, out chan<- interface{}) error {
	atomic.AddInt32(&wt.Count, 1)
	return nil
}

var (
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

	getWorker = func(ctx context.Context, wt workerTest) *gorkers.Runner {
		worker := gorkers.NewRunner(ctx, wt.worker, wt.numWorkers, wt.numWorkers)
		if wt.timeout > 0 {
			worker.SetWorkerTimeout(wt.timeout)
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
	worker      gorkers.WorkFunc
	numWorkers  int64
	testSignal  bool
	errExpected bool
}

type TestWorkerObject struct {
	workFunc func(ctx context.Context, in interface{}, out chan<- interface{}) error
}

func NewTestWorkerObject(wf func(ctx context.Context, in interface{}, out chan<- interface{}) error) gorkers.WorkFunc {
	return wf
}

func (tw *TestWorkerObject) Work(ctx context.Context, in interface{}, out chan<- interface{}) error {
	return tw.workFunc(ctx, in, out)
}

func workBasicNoOut() func(ctx context.Context, in interface{}, out chan<- interface{}) error {
	return func(ctx context.Context, in interface{}, out chan<- interface{}) error {
		_ = in.(int)
		return nil
	}
}

func workBasic() func(ctx context.Context, in interface{}, out chan<- interface{}) error {
	return func(ctx context.Context, in interface{}, out chan<- interface{}) error {
		i := in.(int)
		out <- i
		return nil
	}
}

func workWithError(err error) func(ctx context.Context, in interface{}, out chan<- interface{}) error {
	return func(ctx context.Context, in interface{}, out chan<- interface{}) error {
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
			workerOne := getWorker(ctx, tt)
			// always need a consumer for the out tests so using basic here.
			workerTwo := gorkers.NewRunner(ctx, NewTestWorkerObject(workBasicNoOut()), workerCount, workerCount).InFrom(workerOne)

			if err := workerOne.Start(); err != nil && !tt.errExpected {
				t.Error(err)
			}
			if err := workerTwo.Start(); err != nil && !tt.errExpected {
				t.Error(err)
			}

			for i := 0; i < runTimes; i++ {
				workerOne.Send(i)
			}

			workerOne.Wait().Stop()
			workerTwo.Wait().Stop()
		})
	}
}

func TestWorkersFinish100(t *testing.T) {
	const workCount = 100
	ctx := context.Background()
	w1 := NewWorkerOne()
	w2 := NewWorkerTwo()
	workerOne := gorkers.NewRunner(ctx, w1.Work, 1000, 10)
	workerTwo := gorkers.NewRunner(ctx, w2.Work, 1000, 10000).InFrom(workerOne)
	workerOne.Start()
	workerTwo.Start()

	for i := 0; i < workCount; i++ {
		workerOne.Send(rand.Intn(100))
	}

	workerOne.Wait().Stop()

	workerTwo.Wait().Stop()

	if w1.CurrentCount() != workCount {
		t.Log("worker one failed to finish,", "worker_one count", w1.CurrentCount(), "/", workCount)
		t.Fail()
	}
	if w2.CurrentCount() != workCount {
		t.Log("worker two failed to finish,", "worker_two count", w2.CurrentCount(), "/", workCount)
		t.Fail()
	}

	t.Logf("worker_one count: %d, worker_two count: %d", w1.CurrentCount(), w2.CurrentCount())
}

func TestWorkersFinish100000(t *testing.T) {
	const workCount = 100000
	ctx := context.Background()
	w1 := NewWorkerOne()
	w2 := NewWorkerTwo()
	workerOne := gorkers.NewRunner(ctx, w1.Work, 1000, 2000)
	workerTwo := gorkers.NewRunner(ctx, w2.Work, 1000, 1).InFrom(workerOne)
	workerOne.Start()
	workerTwo.Start()

	for i := 0; i < workCount; i++ {
		workerOne.Send(rand.Intn(100))
	}

	workerOne.Wait().Stop()

	workerTwo.Wait().Stop()

	if w1.CurrentCount() != workCount {
		t.Log("worker one failed to finish,", "worker_one count", w1.CurrentCount(), "/", workCount)
		t.Fail()
	}
	if w2.CurrentCount() != workCount {
		t.Log("worker two failed to finish,", "worker_two count", w2.CurrentCount(), "/", workCount)
		t.Fail()
	}

	t.Logf("worker_one count: %d, worker_two count: %d", w1.CurrentCount(), w2.CurrentCount())
}

func TestWorkersFinish1000000(t *testing.T) {
	const workCount = 1000000
	ctx := context.Background()
	w1 := NewWorkerOne()
	w2 := NewWorkerTwo()
	workerOne := gorkers.NewRunner(ctx, w1.Work, 1000, 1000)
	workerTwo := gorkers.NewRunner(ctx, w2.Work, 1000, 500).InFrom(workerOne)
	workerOne.Start()
	workerTwo.Start()

	for i := 0; i < workCount; i++ {
		workerOne.Send(rand.Intn(100))
	}

	workerOne.Wait().Stop()

	workerTwo.Wait().Stop()

	if w1.CurrentCount() != workCount {
		t.Log("worker one failed to finish,", "worker_one count", w1.CurrentCount(), "/", workCount)
		t.Fail()
	}
	if w2.CurrentCount() != workCount {
		t.Log("worker two failed to finish,", "worker_two count", w2.CurrentCount(), "/", workCount)
		t.Fail()
	}

	t.Logf("worker_one count: %d, worker_two count: %d", w1.CurrentCount(), w2.CurrentCount())
}

func BenchmarkGoWorkers1to1(b *testing.B) {
	worker := gorkers.NewRunner(context.Background(), NewTestWorkerObject(workBasicNoOut()), 1000, 2000)
	worker.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			worker.Send(j)
		}
	}
	b.StopTimer()

	worker.Wait().Stop()
}

func Benchmark100GoWorkers(b *testing.B) {
	b.ReportAllocs()
	worker := gorkers.NewRunner(context.Background(), NewTestWorkerObject(workBasicNoOut()), 100, 200)
	worker.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.Send(i)
	}

	worker.Wait().Stop()
}

func Benchmark1000GoWorkers(b *testing.B) {
	b.ReportAllocs()
	worker := gorkers.NewRunner(context.Background(), NewTestWorkerObject(workBasicNoOut()), 1000, 500)
	worker.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.Send(i)
	}

	worker.Wait().Stop()
}

func Benchmark10000GoWorkers(b *testing.B) {
	b.ReportAllocs()
	worker := gorkers.NewRunner(context.Background(), NewTestWorkerObject(workBasicNoOut()), 10000, 5000)
	worker.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.Send(i)
	}

	worker.Wait().Stop()
}
