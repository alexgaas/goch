package internal

func Generate() chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for i := range 10 {
			ch <- i + 1
		}
	}()

	return ch
}
