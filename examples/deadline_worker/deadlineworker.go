package main

import (
	"context"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"time"
)

func main() {
	ctx := context.Background()
	t := time.Now()

	deadlineWorker := worker.NewWorker(ctx, NewDeadlineWorker(), 4).SetDeadline(t.Add(2 * time.Second)).Work()

	for i := 0; i < 100; i++ {
		deadlineWorker.Send("hello")
	}

	err := deadlineWorker.Close()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("finished")
}

type DeadlineWorker struct{}

func NewDeadlineWorker() *DeadlineWorker {
	return &DeadlineWorker{}
}

func (dlw *DeadlineWorker) Work(w *worker.Worker, in interface{}) error {
	fmt.Println(in)
	time.Sleep(1 * time.Second)
	return nil
}
