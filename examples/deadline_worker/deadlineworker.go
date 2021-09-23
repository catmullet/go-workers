//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/catmullet/go-workers"
)

func main() {
	ctx := context.Background()
	t := time.Now()

	deadlineWorker := workers.NewRunner(ctx, NewDeadlineWorker().Work, 100, 100).
		SetDeadline(t.Add(200 * time.Millisecond))
	deadlineWorker.Start()
	if err := deadlineWorker.Start(); err != nil {
		fmt.Println(err)
	}

	for i := 0; i < 1000000; i++ {
		deadlineWorker.Send("hello")
	}

	deadlineWorker.Wait().Stop()
	fmt.Println("finished")
}

type DeadlineWorker struct{}

func NewDeadlineWorker() *DeadlineWorker {
	return &DeadlineWorker{}
}

func (dlw *DeadlineWorker) Work(_ context.Context, in interface{}, out chan<- interface{}) error {
	fmt.Println(in)
	time.Sleep(1 * time.Second)
	return nil
}
