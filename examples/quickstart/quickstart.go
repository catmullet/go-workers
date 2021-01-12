package main

import (
	"context"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"math/rand"
)

func main() {
	ctx := context.Background()
	w := worker.NewWorker(ctx, NewWorker(), 2).Work()

	for i := 0; i < 1000; i++ {
		w.Send(rand.Intn(100))
	}

	if err := w.Close(); err != nil {
		fmt.Println(err)
	}
}

type Worker struct {
}

func NewWorker() *Worker {
	return &Worker{}
}

func (wo *Worker) Work(_ *worker.Worker, in interface{}) error {
	total := in.(int) * 2
	fmt.Println(fmt.Sprintf("%d * 2 = %d", in.(int), total))
	return nil
}
