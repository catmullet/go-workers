// +build ignore

package main

import (
	"context"
	"fmt"
	"github.com/catmullet/workers"
	"math/rand"
	"time"
)

func main() {
	ctx := context.Background()
	t := time.Now()
	rnr := workers.NewRunner(ctx, NewWorker(), 1000).Start()

	for i := 0; i < 1000000; i++ {
		rnr.Send(rand.Intn(100))
	}

	if err := rnr.Wait(); err != nil {
		fmt.Println(err)
	}

	totalTime := time.Since(t).Milliseconds()
	fmt.Printf("total time %dms\n", totalTime)
}

type WorkerOne struct {
}

func NewWorker() workers.Worker {
	return &WorkerOne{}
}

func (wo *WorkerOne) Work(in interface{}, out chan<- interface{}) error {
	total := in.(int) * 2
	fmt.Println(fmt.Sprintf("%d * 2 = %d", in.(int), total))
	return nil
}
