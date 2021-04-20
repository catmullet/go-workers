// +build ignore

package main

import (
	"context"
	"fmt"
	"github.com/catmullet/go-workers"
	"math/rand"
)

func main() {
	ctx := context.Background()
	workerOne := workers.NewRunner(ctx, NewWorkerOne(2), 100).Start()
	workerTwo := workers.NewRunner(ctx, NewWorkerTwo(4), 100).InFrom(workerOne).Start()

	for i := 0; i < 15; i++ {
		workerOne.Send(rand.Intn(100))
	}

	if err := workerOne.Wait(); err != nil {
		fmt.Println(err)
	}

	if err := workerTwo.Wait(); err != nil {
		fmt.Println(err)
	}
}

type WorkerOne struct {
	amountToMultiply int
}
type WorkerTwo struct {
	amountToMultiply int
}

func NewWorkerOne(amountToMultiply int) workers.Worker {
	return &WorkerOne{
		amountToMultiply: amountToMultiply,
	}
}

func NewWorkerTwo(amountToMultiply int) workers.Worker {
	return &WorkerTwo{
		amountToMultiply,
	}
}

func (wo *WorkerOne) Work(in interface{}, out chan<- interface{}) error {
	total := in.(int) * wo.amountToMultiply
	fmt.Println("worker1", fmt.Sprintf("%d * %d = %d", in.(int), wo.amountToMultiply, total))
	out <- total
	return nil
}

func (wt *WorkerTwo) Work(in interface{}, out chan<- interface{}) error {
	totalFromWorkerOne := in.(int)
	fmt.Println("worker2", fmt.Sprintf("%d * %d = %d", totalFromWorkerOne, wt.amountToMultiply, totalFromWorkerOne*wt.amountToMultiply))
	return nil
}
