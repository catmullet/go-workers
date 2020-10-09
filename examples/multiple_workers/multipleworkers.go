package main

import (
	"context"
	"fmt"
	goworker "github.com/catmullet/go-workers"
	"math/rand"
)

func main() {
	ctx := context.Background()
	workerOne := goworker.NewWorker(ctx, workerFunctionOne, 10).Work()
	workerTwo := goworker.NewWorker(ctx, workerFunctionTwo, 10).InFrom(workerOne).Work()

	for i := 0; i < 10; i++ {
		workerOne.Send(rand.Intn(100))
	}

	workerOne.Close()
	if err := workerOne.Wait(); err != nil {
		fmt.Println(err)
	}

	workerTwo.Close()
	if err := workerTwo.Wait(); err != nil {
		fmt.Println(err)
	}
}

func workerFunctionOne(w *goworker.Worker) error {
	for in := range w.In() {
		total := in.(int) * 2
		fmt.Println(fmt.Sprintf("%d * 2 = %d", in.(int), total))
		w.Out(total)
	}
	return nil
}

func workerFunctionTwo(w *goworker.Worker) error {
	for in := range w.In() {
		totalFromWorkerOne := in.(int)
		fmt.Println(fmt.Sprintf("%d * 4 = %d", totalFromWorkerOne, totalFromWorkerOne*4))
	}
	return nil
}
