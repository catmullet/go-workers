package main

import (
	"context"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"math/rand"
	"time"
)

func main() {
	ctx := context.Background()
	workerOne := worker.NewWorker(ctx, NewWorkerOne(), 10).Work()
	workerTwo := worker.NewWorker(ctx, NewWorkerTwo(), 10).AddBuffer(1000).InFrom(workerOne).Work()

	for i := 0; i < 10000; i++ {
		workerOne.Send(rand.Intn(100))
	}

	workerOneCount := workerOne.WorkerCount() - 10
	workerTwoCount := workerTwo.WorkerCount() - 10

	time.Sleep(1500 * time.Millisecond)

	workerOneCountAfterCooldown := workerOne.WorkerCount() - 10
	workerTwoCountAfterCooldown := workerTwo.WorkerCount() - 10

	if err := workerOne.Close(); err != nil {
		fmt.Println(err)
	}

	if err := workerTwo.Close(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("current workers added to worker one, ", workerOneCount)
	fmt.Println("current workers added to worker two, ", workerTwoCount)

	fmt.Println("worker one count after cooldown, ", workerOneCountAfterCooldown)
	fmt.Println("worker two count after cooldown, ", workerTwoCountAfterCooldown)
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
	fmt.Println(fmt.Sprintf("%d * 2 = %d", in.(int), total))
	w.Out(total)
	return nil
}

func (wt *WorkerTwo) Work(w *worker.Worker, in interface{}) error {
	totalFromWorkerOne := in.(int)
	fmt.Println(fmt.Sprintf("%d * 4 = %d", totalFromWorkerOne, totalFromWorkerOne*4))
	time.Sleep(time.Second)
	return nil
}
