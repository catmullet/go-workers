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
	workerCount   = 100000
	workerTimeout = time.Millisecond * 300
	runTimes      = 100000
)

type WorkerOne struct {
	Count int
	sync.Mutex
}
type WorkerTwo struct {
	Count int
	sync.Mutex
}

func NewWorkerOne() *WorkerOne {
	return &WorkerOne{}
}

func NewWorkerTwo() *WorkerTwo {
	return &WorkerTwo{}
}

func (wo *WorkerOne) CurrentCount() int {
	wo.Lock()
	defer wo.Unlock()
	return wo.Count
}

func (wo *WorkerOne) Work(in interface{}, out chan<- interface{}) error {
	mut.Lock()
	wo.Count = wo.Count + 1
	mut.Unlock()

	total := in.(int) * 2
	out <- total
	return nil
}

func (wt *WorkerTwo) CurrentCount() int {
	wt.Lock()
	defer wt.Unlock()
	return wt.Count
}

func (wt *WorkerTwo) Work(in interface{}, out chan<- interface{}) error {
	mut.Lock()
	wt.Count = wt.Count + 1
	mut.Unlock()
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

			if err := workerOne.Wait(); err != nil && (!tt.errExpected) {
				t.Error(err)
			}
			if err := workerTwo.Wait(); err != nil && !tt.errExpected {
				t.Error(err)
			}
		})
	}
}

func TestWorkersFinish100(t *testing.T) {
	const workCount = 100
	ctx := context.Background()
	w1 := NewWorkerOne()
	w2 := NewWorkerTwo()
	workerOne := NewRunner(ctx, w1, 1000).Start()
	workerTwo := NewRunner(ctx, w2, 1000).InFrom(workerOne).Start()

	for i := 0; i < workCount; i++ {
		workerOne.Send(rand.Intn(100))
	}

	if err := workerOne.Wait(); err != nil {
		fmt.Println(err)
	}

	if err := workerTwo.Wait(); err != nil {
		fmt.Println(err)
	}

	if w1.CurrentCount() != workCount {
		t.Log("worker one failed to finish,", "worker_one count", w1.CurrentCount(), "/ 100000")
		t.Fail()
	}
	if w2.CurrentCount() != workCount {
		t.Log("worker two failed to finish,", "worker_two count", w2.CurrentCount(), "/ 100000")
		t.Fail()
	}

	t.Logf("worker_one count: %d, worker_two count: %d", w1.CurrentCount(), w2.CurrentCount())
}

func TestWorkersFinish100000(t *testing.T) {
	const workCount = 100000
	ctx := context.Background()
	w1 := NewWorkerOne()
	w2 := NewWorkerTwo()
	workerOne := NewRunner(ctx, w1, 1000).Start()
	workerTwo := NewRunner(ctx, w2, 1000).InFrom(workerOne).Start()

	for i := 0; i < workCount; i++ {
		workerOne.Send(rand.Intn(100))
	}

	if err := workerOne.Wait(); err != nil {
		fmt.Println(err)
	}

	if err := workerTwo.Wait(); err != nil {
		fmt.Println(err)
	}

	if w1.CurrentCount() != workCount {
		t.Log("worker one failed to finish,", "worker_one count", w1.CurrentCount(), "/ 100000")
		t.Fail()
	}
	if w2.CurrentCount() != workCount {
		t.Log("worker two failed to finish,", "worker_two count", w2.CurrentCount(), "/ 100000")
		t.Fail()
	}

	t.Logf("worker_one count: %d, worker_two count: %d", w1.CurrentCount(), w2.CurrentCount())
}

func TestWorkersFinish1000000(t *testing.T) {
	const workCount = 1000000
	ctx := context.Background()
	w1 := NewWorkerOne()
	w2 := NewWorkerTwo()
	workerOne := NewRunner(ctx, w1, 1000).Start()
	workerTwo := NewRunner(ctx, w2, 1000).InFrom(workerOne).Start()

	for i := 0; i < workCount; i++ {
		workerOne.Send(rand.Intn(100))
	}

	if err := workerOne.Wait(); err != nil {
		fmt.Println(err)
	}

	if err := workerTwo.Wait(); err != nil {
		fmt.Println(err)
	}

	if w1.CurrentCount() != workCount {
		t.Log("worker one failed to finish,", "worker_one count", w1.CurrentCount(), "/ 100000")
		t.Fail()
	}
	if w2.CurrentCount() != workCount {
		t.Log("worker two failed to finish,", "worker_two count", w2.CurrentCount(), "/ 100000")
		t.Fail()
	}

	t.Logf("worker_one count: %d, worker_two count: %d", w1.CurrentCount(), w2.CurrentCount())
}

func BenchmarkGoWorkers1to1(b *testing.B) {
	worker := NewRunner(context.Background(), NewTestWorkerObject(workBasicNoOut()), 1000).Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			worker.Send(j)
		}
	}
	b.StopTimer()

	if err := worker.Wait(); err != nil {
		b.Error(err)
	}
}

func Benchmark100GoWorkers(b *testing.B) {
	b.ReportAllocs()
	worker := NewRunner(context.Background(), NewTestWorkerObject(workBasicNoOut()), 100).Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.Send(i)
	}

	if err := worker.Wait(); err != nil {
		b.Error(err)
	}
}

func Benchmark1000GoWorkers(b *testing.B) {
	b.ReportAllocs()
	worker := NewRunner(context.Background(), NewTestWorkerObject(workBasicNoOut()), 1000).Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.Send(i)
	}

	if err := worker.Wait(); err != nil {
		b.Error(err)
	}
}

func Benchmark10000GoWorkers(b *testing.B) {
	b.ReportAllocs()
	worker := NewRunner(context.Background(), NewTestWorkerObject(workBasicNoOut()), 10000).Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.Send(i)
	}

	if err := worker.Wait(); err != nil {
		b.Error(err)
	}
}
