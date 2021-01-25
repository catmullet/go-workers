// +build ignore

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
	t := time.Now()
	w := worker.NewWorker(ctx, NewWorker(), 1000).Work()

	for i := 0; i < 1000000; i++ {
		w.Send(rand.Intn(100))
	}

	if err := w.Close(); err != nil {
		fmt.Println(err)
	}

	totalTime := time.Since(t).Milliseconds()
	fmt.Printf("total time %dms\n", totalTime)
}

type Worker struct {
}

func NewWorker() *Worker {
	return &Worker{}
}

func (wo *Worker) Work(w *worker.Worker, in interface{}) error {
	total := in.(int) * 2
	defer w.Println(fmt.Sprintf("%d * 2 = %d", in.(int), total))
	return nil
}
