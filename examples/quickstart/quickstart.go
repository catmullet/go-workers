package main

import (
	"context"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"github.com/pkg/profile"
	"math/rand"
	"time"
)

func main() {
	ctx := context.Background()

	defer profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop()
	w := worker.NewWorker(ctx, NewWorker(), 1000).Work()

	t := time.Now()
	for i := 0; i < 100000; i++ {
		w.Send(rand.Intn(100))
	}

	if err := w.Close(); err != nil {
		fmt.Println(err)
	}
	totalTime := time.Since(t).Milliseconds()
	fmt.Println(fmt.Sprintf("total time %dms", totalTime))
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
