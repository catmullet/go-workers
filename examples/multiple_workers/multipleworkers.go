//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/guilhem/gorkers"
)

var (
	count = make(map[string]int)
	mut   = sync.RWMutex{}
)

func main() {
	ctx := context.Background()

	workerOne := gorkers.NewRunner(ctx, NewWorkerOne().Work, 1000, 1000)
	workerTwo := gorkers.NewRunner(ctx, NewWorkerTwo().Work, 1000, 1000).InFrom(workerOne)
	if err := workerOne.Start(); err != nil {
		fmt.Println(err)
	}
	if err := workerTwo.Start(); err != nil {
		fmt.Println(err)
	}

	go func() {
		for i := 0; i < 100000; i++ {
			workerOne.Send(rand.Intn(100))
		}
		if err := workerOne.Wait().Stop(); err != nil {
			fmt.Println(err)
		}
	}()

	if err := workerTwo.Wait().Stop(); err != nil {
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

func NewWorkerOne() *WorkerOne {
	return &WorkerOne{}
}

func NewWorkerTwo() *WorkerTwo {
	return &WorkerTwo{}
}

func (wo *WorkerOne) Work(_ context.Context, in interface{}, out chan<- interface{}) error {
	var workerOne = "worker_one"
	mut.Lock()
	if val, ok := count[workerOne]; ok {
		count[workerOne] = val + 1
	} else {
		count[workerOne] = 1
	}
	mut.Unlock()

	total := in.(int) * 2
	fmt.Println("worker1", fmt.Sprintf("%d * 2 = %d", in.(int), total))
	out <- total
	return nil
}

func (wt *WorkerTwo) Work(_ context.Context, in interface{}, out chan<- interface{}) error {
	var workerTwo = "worker_two"
	mut.Lock()
	if val, ok := count[workerTwo]; ok {
		count[workerTwo] = val + 1
	} else {
		count[workerTwo] = 1
	}
	mut.Unlock()

	totalFromWorkerOne := in.(int)
	fmt.Println("worker2", fmt.Sprintf("%d * 4 = %d", totalFromWorkerOne, totalFromWorkerOne*4))
	return nil
}
