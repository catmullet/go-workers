//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/guilhem/gorkers"
)

func main() {
	ctx := context.Background()

	timeoutWorker := gorkers.NewRunner(ctx, work, 10, 10).SetWorkerTimeout(100 * time.Millisecond)
	err := timeoutWorker.Start()
	if err != nil {
		fmt.Println(err)
	}

	for i := 0; i < 1000000; i++ {
		timeoutWorker.Send("hello")
	}

	timeoutWorker.Wait().Stop()
}

func work(ctx context.Context, in interface{}, out chan<- interface{}) error {
	fmt.Println(in)
	time.Sleep(1 * time.Second)
	return nil
}
