package internal

import (
	"context"
	"sync"
)

func WorkerPool(ctx context.Context, in chan int, numWorkers int) chan int {
	return pool(ctx, in, wrapper, numWorkers)
}

func pool(ctx context.Context, in chan int, f func(context.Context, int) int, numWorkers int) chan int {
	out := make(chan int)
	wg := &sync.WaitGroup{}
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case v, ok := <-in:
					if !ok {
						return
					}
					select {
					case out <- f(ctx, v):
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(out)
		wg.Wait()
	}()

	return out
}
