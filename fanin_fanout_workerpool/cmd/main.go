package main

import (
	"context"
	. "fanin_fanout_workerpool/internal"
	"fmt"
	"time"
)

/*
implement function to collect data from array of channels into one channel. make a context cancellation
*/
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startFanInFanOut := time.Now()
	counter := NewCounter()
	for v := range FanIn(ctx, Fanout(ctx, Generate(), 10)) {
		fmt.Println("fin/fout: ", v)
	}

	counter.Stop()
	fanInFanOutGCount := counter.GetCount()

	// make worker pool
	startWorkerPool := time.Now()
	counter.Reset()
	for v := range WorkerPool(ctx, Generate(), 10) {
		fmt.Println("pool: ", v)
	}

	counter.Stop()
	workerPoolGCount := counter.GetCount()

	fmt.Println()
	fmt.Println("total time for fan-in / fan-out in ms:", time.Since(startFanInFanOut).Milliseconds())
	fmt.Println("number of goroutines for fan-in / fan-out:", fanInFanOutGCount)
	fmt.Println()
	fmt.Println("total time for worker pool", time.Since(startWorkerPool).Milliseconds())
	fmt.Println("number of goroutines for worker pool in ms", workerPoolGCount)

	fmt.Println("done...")
}
