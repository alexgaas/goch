package main

import (
	"fmt"
	"sync"
)

func generate() <-chan int {
	ch := make(chan int)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 10 {
			ch <- i
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 10 {
			ch <- i + 10
		}
	}()

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}

/*
Create channel and create M goroutine (2,3,4...) writing into channel, then create N different goroutines (2,3,4...) reading from the channel.
*/
func main() {
	ch := generate()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for v := range ch {
			fmt.Println("reading g1: ", v)
		}
	}()

	go func() {
		defer wg.Done()
		for v := range ch {
			fmt.Println("reading g2: ", v)
		}
	}()

	wg.Wait()
}
