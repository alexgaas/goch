package internal

import (
	"context"
	"math/rand"
	"time"
)

func pipeline(ctx context.Context, in chan int, f func(context.Context, int) int) chan int {
	ch := make(chan int)

	go func() {
		defer close(ch)
		for {
			select {
			case v, ok := <-in:
				if !ok {
					return
				}
				if v == -1 {
					return
				}
				select {
				case ch <- f(ctx, v):
				case <-ctx.Done():
					//fmt.Println("timed out")
					return
				}
			case <-ctx.Done():
				//fmt.Println("timed out")
				return
			}
		}
	}()

	return ch
}

func wrapper(ctx context.Context, v int) int {
	ch := make(chan struct{})

	go func() {
		defer close(ch)
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		ch <- struct{}{}
	}()

	select {
	case <-ch:
		return square(v)
	case <-ctx.Done():
		//fmt.Println("timed out")
		return -1
	}
}

func square(v int) int {
	return v * v
}
