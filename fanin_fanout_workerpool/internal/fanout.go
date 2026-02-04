package internal

import (
	"context"
)

func Fanout(ctx context.Context, in chan int, numWorkers int) []chan int {
	chans := make([]chan int, numWorkers)

	for i := range chans {
		chans[i] = pipeline(ctx, in, wrapper)
	}

	return chans
}
