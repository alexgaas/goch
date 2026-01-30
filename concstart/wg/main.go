package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

/*
you have a function running between 1 and N seconds. Run this function concurrently M times and print out how many seconds runs main and how many seconds run all functions in parallel.
*/

// run between 1 and 5s
func randomRun() int {
	runTime := rand.Intn(1 + 5)
	time.Sleep(time.Duration(runTime) * time.Second)
	return runTime
}

func main() {
	start := time.Now()

	var runTime int

	mu := sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(100)
	// let's run 100 times
	for range 100 {
		go func() {
			defer wg.Done()
			t := randomRun()

			mu.Lock()
			runTime += t
			mu.Unlock()
		}()
	}
	wg.Wait()

	// should be around 5
	fmt.Println("main time:", time.Since(start).Seconds())
	// should be btw 100 and 500
	fmt.Println("total work seconds:", runTime)
}
