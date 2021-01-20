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
	newWorker := NewWorker()
	w := worker.NewWorker(ctx, newWorker, 100).AddCloser(newWorker.Closer).Work()

	for i := 0; i < 1000000; i++ {
		w.Send(rand.Intn(100))
	}
	if err := w.Close(); err != nil {
		fmt.Println(err)
	}
	since := time.Since(t)
	fmt.Println(since.Milliseconds(), "ms")
}

type Worker struct {
}

func (wrkr *Worker) Closer(w *worker.Worker) error {
	w.Println("closing up shop with the closer")
	return nil
}

func NewWorker() *Worker {
	return &Worker{}
}

func (wo *Worker) Work(w *worker.Worker, in interface{}) error {
	total := in.(int) * 2
	w.Println(fmt.Sprintf("%d * 2 = %d", in.(int), total))
	return nil
}
