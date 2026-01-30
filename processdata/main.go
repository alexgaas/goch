package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

/*
you have a function to process data which for simplicity takes an integer number K and returns K * 2 after some wait (let's say it awaits between 1 and 10 seconds).
write data (10 numbers as example) into some buffer concurrently and then process data in parallel with N number of workers.
*/
func work(k int) int {
	t := rand.Intn(10)
	time.Sleep(time.Duration(t) * time.Second)
	return k * 2
}

func processData(in <-chan int, out chan<- int, numWorkers int) {
	wg := &sync.WaitGroup{}
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for v := range in {
				out <- work(v)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()
}

func main() {
	in := make(chan int)
	out := make(chan int)
	go func() {
		for i := range 10 {
			in <- i
		}
		close(in)
	}()

	start := time.Now()
	processData(in, out, 5)

	for v := range out {
		fmt.Println(v)
	}

	fmt.Println(time.Since(start))
}
