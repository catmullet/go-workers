// +build ignore

package main

import (
	"context"
	"fmt"
	"github.com/catmullet/go-workers"
	"math/rand"
	"sync"
)

var (
	count = make(map[string]int)
	mut   = sync.RWMutex{}
)

func main() {
	ctx := context.Background()

	workerOne := workers.NewRunner(ctx, NewWorkerOne(), 1000).Start()
	workerTwo := workers.NewRunner(ctx, NewWorkerTwo(), 1000).InFrom(workerOne).Start()

	for i := 0; i < 100000; i++ {
		workerOne.Send(rand.Intn(100))
	}

	if err := workerOne.Wait(); err != nil {
		fmt.Println(err)
	}

	if err := workerTwo.Wait(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("worker_one", count["worker_one"])
	fmt.Println("worker_two", count["worker_two"])
	fmt.Println("finished")
}

type WorkerOne struct {
}
type WorkerTwo struct {
}

func NewWorkerOne() workers.Worker {
	return &WorkerOne{}
}

func NewWorkerTwo() workers.Worker {
	return &WorkerTwo{}
}

func (wo *WorkerOne) Work(in interface{}, out chan<- interface{}) error {
	var workerOne = "worker_one"
	mut.Lock()
	defer mut.Unlock()
	if val, ok := count[workerOne]; ok {
		count[workerOne] = val + 1
	} else {
		count[workerOne] = 1
	}

	total := in.(int) * 2
	fmt.Println("worker1", fmt.Sprintf("%d * 2 = %d", in.(int), total))
	out <- total
	return nil
}

func (wt *WorkerTwo) Work(in interface{}, out chan<- interface{}) error {
	var workerTwo = "worker_two"
	mut.Lock()
	defer mut.Unlock()
	if val, ok := count[workerTwo]; ok {
		count[workerTwo] = val + 1
	} else {
		count[workerTwo] = 1
	}

	totalFromWorkerOne := in.(int)
	fmt.Println("worker2", fmt.Sprintf("%d * 4 = %d", totalFromWorkerOne, totalFromWorkerOne*4))
	return nil
}
