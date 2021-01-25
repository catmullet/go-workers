// +build ignore

package main

import (
	"context"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"math/rand"
)

func main() {
	ctx := context.Background()
	workerOne := worker.NewWorker(ctx, NewWorkerOne(), 1000).Work()
	workerTwo := worker.NewWorker(ctx, NewWorkerTwo(), 1000).InFrom(workerOne).Work()

	for i := 0; i < 1000000; i++ {
		workerOne.Send(rand.Intn(100))
	}

	if err := workerOne.Close(); err != nil {
		fmt.Println(err)
	}

	if err := workerTwo.Close(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("finished")
}

type WorkerOne struct{}
type WorkerTwo struct{}

func NewWorkerOne() *WorkerOne {
	return &WorkerOne{}
}

func NewWorkerTwo() *WorkerTwo {
	return &WorkerTwo{}
}

func (wo *WorkerOne) Work(w *worker.Worker, in interface{}) error {
	total := in.(int) * 2
	w.Println(fmt.Sprintf("%d * 2 = %d", in.(int), total))
	w.Out(total)
	return nil
}

func (wt *WorkerTwo) Work(w *worker.Worker, in interface{}) error {
	totalFromWorkerOne := in.(int)
	w.Println(fmt.Sprintf("%d * 4 = %d", totalFromWorkerOne, totalFromWorkerOne*4))
	return nil
}
