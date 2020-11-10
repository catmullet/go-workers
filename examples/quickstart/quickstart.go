package main

import (
	"context"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"math/rand"
)

func main() {
	ctx := context.Background()
	w := worker.NewWorker(ctx, NewWorker(), 10).Work()

	for i := 0; i < 10; i++ {
		w.Send(rand.Intn(100))
	}

	w.Close()
	err := w.Wait()

	if err != nil {
		fmt.Println(err)
	}
}

type Worker struct{}

func NewWorker() *Worker {
	return &Worker{}
}

func (wo *Worker) Work(w *worker.Worker) error {
	for in := range w.In() {
		total := in.(int) * 2
		fmt.Println(fmt.Printf("%d * 2 = %d", in.(int), total))
	}
	return nil
}
