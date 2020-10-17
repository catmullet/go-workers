package main

import (
	"context"
	"errors"
	"fmt"
	worker "github.com/catmullet/go-workers"
	"time"
)

func main() {
	ctx := context.Background()

	newWorker := worker.NewWorker(ctx, timeoutWorkerFunction, 100).Work()

	for i := 0; i < 10000; i++ {
		newWorker.Send(fmt.Sprintf("hello, %d", i))
	}

	newWorker.Close()
	err := newWorker.Wait()

	if err != nil {
		fmt.Println(err)
	}
}

func timeoutWorkerFunction(w *worker.Worker) error {
	timer := time.NewTimer(10 * time.Second)
	var err error

	for {
		select {
		case <-timer.C:
			err = errors.New("timed out")
			return err
		case in := <-w.In():
			if in != nil {
				fmt.Println(in)
			} else {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}
