package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

/*
You have a function to process data which for simplicity takes an integer number K and returns K * 2 after some wait (let's say it awaits between 0 and 10 seconds).
write data (10 numbers as example) into some buffer concurrently and then process data in parallel with N number of workers, every process should run no longer than M seconds (5 by example) and print out time of execution.
Please implement possible cancellation with timeout context in process data function.
*/
func work(ctx context.Context, k int) int {
	ch := make(chan struct{})

	go func() {
		t := rand.Intn(10)
		time.Sleep(time.Duration(t) * time.Second)
		close(ch)
	}()

	select {
	case <-ch:
		return k * 2
	case <-ctx.Done():
		return 0
	}
}

func processData(ctx context.Context, in <-chan int, out chan<- int, numWorkers int) {
	wg := &sync.WaitGroup{}
	for range numWorkers {
		wg.Add(1)
		go worker(ctx, wg, in, out)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
}

func worker(ctx context.Context, wg *sync.WaitGroup, in <-chan int, out chan<- int) {
	defer wg.Done()
	for /*v := range in*/ {
		select {
		case v, ok := <-in:
			if !ok {
				return
			}
			select {
			case out <- work(ctx, v):
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	in := make(chan int)
	out := make(chan int)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		// simulate delay on writer
		// time.Sleep(10 * time.Second)
		for i := range 10 {
			select {
			case in <- i:
				in <- i + 1
			case <-ctx.Done():
				return
			}
		}
		close(in)
	}()

	start := time.Now()
	processData(ctx, in, out, 5)

	for v := range out {
		fmt.Println(v)
	}

	fmt.Println(time.Since(start))
}
