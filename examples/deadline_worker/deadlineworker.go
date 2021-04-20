// +build ignore

package main

import (
	"context"
	"fmt"
	"github.com/catmullet/workers"
	"time"
)

func main() {
	ctx := context.Background()
	t := time.Now()

	deadlineWorker := workers.NewRunner(ctx, NewDeadlineWorker(), 100).
		SetDeadline(t.Add(200 * time.Millisecond)).Start()

	for i := 0; i < 1000000; i++ {
		deadlineWorker.Send("hello")
	}

	err := deadlineWorker.Wait()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("finished")
}

type DeadlineWorker struct{}

func NewDeadlineWorker() workers.Worker {
	return &DeadlineWorker{}
}

func (dlw *DeadlineWorker) Work(in interface{}, out chan<- interface{}) error {
	fmt.Println(in)
	time.Sleep(1 * time.Second)
	return nil
}
