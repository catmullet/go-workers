// +build ignore

package main

import (
	"context"
	"fmt"
	"github.com/catmullet/go-workers"
	"time"
)

func main() {
	ctx := context.Background()

	timeoutWorker := workers.NewRunner(ctx, NewTimeoutWorker(), 10).SetTimeout(100 * time.Millisecond).Start()

	for i := 0; i < 1000000; i++ {
		timeoutWorker.Send("hello")
	}

	err := timeoutWorker.Wait()
	if err != nil {
		fmt.Println(err)
	}
}

type TimeoutWorker struct{}

func NewTimeoutWorker() workers.Worker {
	return &TimeoutWorker{}
}

func (tw *TimeoutWorker) Work(in interface{}, out chan<- interface{}) error {
	fmt.Println(in)
	time.Sleep(1 * time.Second)
	return nil
}
