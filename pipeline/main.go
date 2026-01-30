package main

import (
	"fmt"
	"time"
)

/*
make 3 functions: _writer_ - generates numbers from 1 to 10, _doubler_ - multiplies numbers on 2 with sleep of 500ms simulating some IO-bound work,
_reader_ - reads and prints out on the screen. All functions must be implemented concurrent way and synced accordingly.
*/

func writer() <-chan int {
	ch := make(chan int)

	go func() {
		for i := range 10 {
			ch <- i
		}
		close(ch)
	}()

	return ch
}

func doubler(ch <-chan int) chan int {
	transformCH := make(chan int)
	go func() {
		for v := range ch {
			transformCH <- v * 2
			time.Sleep(time.Duration(500) * time.Millisecond)
		}
		close(transformCH)
	}()

	return transformCH
}

func reader(ch <-chan int) {
	for v := range ch {
		fmt.Println(v)
	}
}

func main() {
	// fork / join (or parallel divide/conq)
	// this is pipeline pattern
	writerCh := writer()
	transformCh := doubler(writerCh)
	reader(transformCh)

	fmt.Println()
	fmt.Println("done...")
}
