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
	workerOne := worker.NewWorker(ctx, NewWorkerOne(2), 10).
		Work()
	workerTwo := worker.NewWorker(ctx, NewWorkerTwo(4), 10).
		InFrom(workerOne).
		Work()

	for i := 0; i < 10; i++ {
		workerOne.Send(rand.Intn(100))
	}

	if err := workerOne.Close(); err != nil {
		fmt.Println(err)
	}

	if err := workerTwo.Close(); err != nil {
		fmt.Println(err)
	}
}

type WorkerOne struct {
	amountToMultiply int
}
type WorkerTwo struct {
	amountToMultiply int
}

func NewWorkerOne(amountToMultiply int) *WorkerOne {
	return &WorkerOne{
		amountToMultiply: amountToMultiply,
	}
}

func NewWorkerTwo(amountToMultiply int) *WorkerTwo {
	return &WorkerTwo{
		amountToMultiply,
	}
}

func (wo *WorkerOne) Work(w *worker.Worker, in interface{}) error {
	total := in.(int) * wo.amountToMultiply
	fmt.Println(fmt.Sprintf("%d * %d = %d", in.(int), wo.amountToMultiply, total))
	w.Out(total)
	return nil
}

func (wt *WorkerTwo) Work(w *worker.Worker, in interface{}) error {
	totalFromWorkerOne := in.(int)
	fmt.Println(fmt.Sprintf("%d * %d = %d", totalFromWorkerOne, wt.amountToMultiply, totalFromWorkerOne*wt.amountToMultiply))
	return nil
}
