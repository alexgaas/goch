package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

/*
we have a function with undefined behavior - it can work in the range between 1 and 100 seconds. make a wrapper for this function
to break execution if it takes more than 3 seconds and return error.
*/

func longWorker() int {
	secs := rand.Intn(100)
	time.Sleep(time.Duration(secs) * time.Second)
	return secs
}

func wrapper() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ch := make(chan int)
	go func() {
		spent := longWorker()
		ch <- spent
		close(ch)
	}()

	select {
	case v := <-ch:
		//fmt.Println("spent: ", v)
		return v, nil
	case <-ctx.Done():
		//fmt.Println("timeout")
		return -1, fmt.Errorf("timeout on %d seconds", 3)
	}
}

func main() {
	out, err := wrapper()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("function time spent: ", out)
	}
}
