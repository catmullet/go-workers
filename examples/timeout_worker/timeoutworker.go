package main

import (
	"context"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"time"
)

func main() {
	ctx := context.Background()

	timeoutWorker := worker.NewWorker(ctx, NewTimeoutWorker(), 4).SetTimeout(2 * time.Second).Work()

	for i := 0; i < 100; i++ {
		timeoutWorker.Send("hello")
	}

	err := timeoutWorker.Wait()
	if err != nil {
		fmt.Println(err)
	}
}

type TimeoutWorker struct{}

func NewTimeoutWorker() *TimeoutWorker {
	return &TimeoutWorker{}
}

func (tw *TimeoutWorker) Work(w *worker.Worker, in interface{}) error {
	fmt.Println(in)
	time.Sleep(1 * time.Second)
	return nil
}
