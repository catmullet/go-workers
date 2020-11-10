package main

import (
	"context"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"math/rand"
)

func main() {
	ctx := context.Background()
	workerOne := worker.NewWorker(ctx, NewWorkerOne(), 10).Work()
	workerTwo := worker.NewWorker(ctx, NewWorkerTwo(), 10).InFrom(workerOne).Work()

	for i := 0; i < 10; i++ {
		workerOne.Send(rand.Intn(100))
	}

	workerOne.Close()
	if err := workerOne.Wait(); err != nil {
		fmt.Println(err)
	}

	workerTwo.Close()
	if err := workerTwo.Wait(); err != nil {
		fmt.Println(err)
	}
}

type WorkerOne struct{}
type WorkerTwo struct{}

func NewWorkerOne() *WorkerOne {
	return &WorkerOne{}
}

func NewWorkerTwo() *WorkerTwo {
	return &WorkerTwo{}
}

func (wo *WorkerOne) Work(w *worker.Worker) error {
	for in := range w.In() {
		total := in.(int) * 2
		fmt.Println(fmt.Printf("%d * 2 = %d", in.(int), total))
		w.Out(total)
	}
	return nil
}

func (wt *WorkerTwo) Work(w *worker.Worker) error {
	for in := range w.In() {
		totalFromWorkerOne := in.(int)
		fmt.Println(fmt.Printf("%d * 4 = %d", totalFromWorkerOne, totalFromWorkerOne*4))
	}
	return nil
}
