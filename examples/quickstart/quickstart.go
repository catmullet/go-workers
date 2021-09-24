//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/guilhem/gorkers"
)

func main() {
	ctx := context.Background()
	t := time.Now()
	rnr := gorkers.NewRunner(ctx, work, 100, 100)

	if err := rnr.Start(); err != nil {
		fmt.Println(err)
	}

	for i := 0; i < 1000000; i++ {
		rnr.Send(rand.Intn(100))
	}

	rnr.Wait().Stop()

	totalTime := time.Since(t).Milliseconds()
	fmt.Printf("total time %dms\n", totalTime)
}

func work(ctx context.Context, in interface{}, out chan<- interface{}) error {
	total := in.(int) * 2
	fmt.Println(fmt.Sprintf("%d * 2 = %d", in.(int), total))
	return nil
}
