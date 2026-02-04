package internal

import (
	"runtime"
	"time"
)

var (
	timerDuration = 1 * time.Millisecond
)

type Counter struct {
	ticker     *time.Ticker
	ticks      int
	Goroutines int
}

func NewCounter() *Counter {
	c := &Counter{
		ticker: time.NewTicker(timerDuration),
	}

	go func() {
		for {
			select {
			case <-c.ticker.C:
				c.ticks++
				c.Goroutines += runtime.NumGoroutine()
			}
		}
	}()

	return c
}

func (c *Counter) Reset() {
	c.ticks = 0
	c.Goroutines = 0
	c.ticker.Reset(timerDuration)
}

func (c *Counter) Stop() {
	c.ticker.Stop()
}

func (c *Counter) GetCount() int {
	if c.ticks == 0 {
		return 0
	}
	return c.Goroutines / c.ticks
}
