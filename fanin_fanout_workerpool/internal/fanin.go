package internal

import (
	"context"
	"sync"
)

func FanIn(ctx context.Context, chans []chan int) chan int {
	ch := make(chan int)

	go func() {
		defer close(ch)
		wg := &sync.WaitGroup{}
		for _, in := range chans {
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
						case ch <- v:
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
		}
		wg.Wait()
	}()

	return ch
}
