package main

import "fmt"

func main() {
	ch := make(chan int)

	/* WRITE */

	// g1
	go func() {
		for i := range 10 * 10 * 10 {
			ch <- i
		}
		close(ch)
	}()

	// g2
	go func() {
		for i := range 10 * 10 * 10 {
			ch <- i * 7
		}
		close(ch)
	}()

	/* READ */

	// g3
	go func() {
		for v := range ch {
			fmt.Println("worker 1: ", v)
		}
	}()

	// g4
	go func() {
		for v := range ch {
			fmt.Println("worker 2: ", v)
		}
	}()

	for v := range ch {
		fmt.Println("worker 3: ", v)
	}
	
	fmt.Println()
	fmt.Println("done...")
}
