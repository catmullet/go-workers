package main

import (
	"context"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"time"
)

func main() {
	ctx := context.Background()
	t := time.Now()

	deadlineWorker := worker.NewWorker(ctx, workerFunction, 4).SetDeadline(t.Add(2 * time.Second)).Work()

	for i := 0; i < 100; i++ {
		deadlineWorker.Send("hello")
	}

	err := deadlineWorker.Wait()
	if err != nil {
		fmt.Println(err)
	}
}

func workerFunction(w *worker.Worker) error {
	for {
		select {
		case in := <-w.In():
			fmt.Println(in)
			time.Sleep(1 * time.Second)
		case <-w.IsDone():
			// due to the nature of errgroups in order to stop the worker from
			// waiting an error needs to be returned
			return fmt.Errorf("deadline reached")
		}
	}
}
