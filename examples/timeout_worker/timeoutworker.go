// +build ignore

package main

import (
	"context"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"time"
)

func main() {
	ctx := context.Background()

	timeoutWorker := worker.NewWorker(ctx, NewTimeoutWorker(), 10).Work()
	timeoutWorker.SetTimeout(100 * time.Millisecond)

	for i := 0; i < 1000000; i++ {
		timeoutWorker.Send("hello")
	}

	err := timeoutWorker.Close()
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
