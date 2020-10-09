package main

import (
	"context"
	"fmt"
	goworker "github.com/catmullet/go-workers"
	"math/rand"
)

func main() {
	ctx := context.Background()
	worker := goworker.NewWorker(ctx, workerFunction, 10).Work()

	for i := 0; i < 10; i++ {
		worker.Send(rand.Intn(100))
	}

	worker.Close()
	err := worker.Wait()

	if err != nil {
		fmt.Println(err)
	}
}

func workerFunction(w *goworker.Worker) error {
	for in := range w.In() {
		total := in.(int) * 2
		fmt.Println(fmt.Sprintf("%d * 2 = %d", in.(int), total))
	}
	return nil
}
