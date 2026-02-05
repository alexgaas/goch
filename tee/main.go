package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

/*
Implement "tee" pattern - function what has one channel as input and N (same as an input one) channels as output. Please implement possible cancellation with timeout context.
*/

func generate() <-chan int {
	ch := make(chan int)

	go func() {
		defer close(ch)
		for i := range 10 {
			ch <- i + 1
		}
	}()

	return ch
}

func tee(ctx context.Context, in <-chan int, numChans int) []chan int {
	out := make([]chan int, numChans)
	for i := 0; i < numChans; i++ {
		out[i] = make(chan int, numChans)
	}

	go func() {
		defer func() {
			for i := 0; i < numChans; i++ {
				close(out[i])
			}
		}()

		for {
			go func() {
				select {
				case v, ok := <-in:
					if !ok {
						return
					}
					wg := &sync.WaitGroup{}
					for i := 0; i < numChans; i++ {
						wg.Add(1)
						go func() {
							defer wg.Done()

							select {
							case out[i] <- v:
							case <-ctx.Done():
								return
							}
						}()
					}

					wg.Wait()
				case <-ctx.Done():
					return
				}
			}()
		}
	}()
	return out
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out := tee(ctx, generate(), 2)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for v := range out[0] {
			fmt.Println("metrics:", v)
		}
	}()

	go func() {
		defer wg.Done()
		for v := range out[1] {
			fmt.Println("counters:", v)
		}
	}()

	wg.Wait()
}
