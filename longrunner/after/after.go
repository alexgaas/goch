package main

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

/*
we have a function with undefined behavior - it can work in the range between 1 and 100 seconds. make a wrapper for this function
to break execution if it takes more than 3 seconds and return error.
*/

func longWorker() {
	secs := rand.Intn(100)
	time.Sleep(time.Duration(secs) * time.Second)
}

func wrapper() error {
	ch := make(chan struct{})

	go func() {
		longWorker()
		close(ch)
	}()

	select {
	case <-ch:
		return nil
		//fmt.Println("function passed")
	case <-time.After(3 * time.Second):
		//fmt.Println("timeout")
		return errors.New("timeout")
	}
}

func main() {
	err := wrapper()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("done")
}
