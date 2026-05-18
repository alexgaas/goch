# Go Channels cheatsheet
This is a really simple cheatsheet to understand how to correctly use channels with goroutines in Go. This basic introduction does not assume 
consideration of Go channels under the hood and instead focuses on the practical aspects of using channels in production-ready services.

## Go concurrency bare minimum
Let's consider a simple example of a program with a goroutine:
```go
func main() {
	go func() { // g1
		fmt.Println("test")
	}()
	fmt.Println("Hello World")
}
```
If we run this program, we would never see the goroutine start because the program exits before the goroutine is really joined to _g0_: 

```ascii
        fork->
              \
g0(main) ----> go g1 ----> println ----> exit
                \
                g1 -----------------------------> println
```
That happens because we do not control when goroutines start in the program - the Go scheduler does it for us (some details about how the scheduler works can be found in ref [2]).

----
Note: If we want any concurrent program in Go to run correctly, we have to:
- synchronize goroutines
- prevent race conditions / contention
- address any possible goroutine leaks (will be considered in a different topic)
----

### synchronize goroutines

The Go concurrency model is based on the **Fork/Join** approach (it is also called parallel divide and conquer - see ref [1] below). To address the problem we had before and start our goroutine, we need to add a **Join** point into our program - i.e., synchronize goroutines. 
```ascii
        fork->
              \
g0(main) ----> go g1 --------> println ----> exit
                \ -> join ^ 
                g1 ----> println
```
How would we do that synchronization (add a join point) in Go? We can add a **wait group**:
```go
func g1(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("test")
}

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go g1(wg)
	wg.Wait()
	fmt.Println("Hello World")
}
```
`WaitGroup` is a **counting semaphore** typically used to wait for a group of goroutines to finish (see [3]).

The result of this execution will be:
```ascii
test
Hello World
```

----

Note: please make sure you pass the `WaitGroup` into functions by reference. If you pass by value, the wait group will be copied and will never wait for the goroutine to end!

----

### prevent race / contention

Let's consider a contention problem when N goroutines are trying to access and modify a single variable in memory.
```go
func main() {
	m := 0
	wg := &sync.WaitGroup{}
	wg.Add(1000)
	for range 1000 {
		go func() {
			defer wg.Done()
			m++
		}()
	}
	wg.Wait()
	fmt.Println(m)
}
```
This program tries to increment one variable with N goroutines (1000 in our case). If we run the program, we'd get the following result:
```ascii
927
...
898
...
946
```
but not 1000 as expected. The reason is that `m = m + 1` is not an atomic operation; it consists of three operations: read m -> add one -> write m. 

flow diagram:
```ascii
main g0 \ -> m = 0
        go g1() -> r/m/w = 0/1/1
        go g2() -> r/m/w = 0/1/1
        /
m = 1, m != 2 as expected
```
That is a classic case of data contention (or race). How would we address this problem? The simplest way would be to use **atomic** types:
```go
func main() {
	var m atomic.Int32
	wg := &sync.WaitGroup{}
	wg.Add(1000)
	for range 1000 {
		go func() {
			defer wg.Done()
			m.Add(1)
		}()
	}
	wg.Wait()
	fmt.Println(m.Load())
}
```
result:
```ascii
1000
```
Another option would be to use a _Mutex_ - a mutual exclusion lock.
```go
func main() {
	m := 0
	wg := &sync.WaitGroup{}
	wg.Add(1000)

	mu := sync.Mutex{}
	for range 1000 {
		go func() {
			defer wg.Done()
			mu.Lock()
			m++
			mu.Unlock()
		}()
	}
	wg.Wait()
	fmt.Println(m)
}
```
result:
```ascii
1000
```
The advantage of a mutex is obvious - at any given moment, only one goroutine may execute the protected set of instructions for you (not only for certain atomic types!).

----

Note: If you have multiple goroutines with separate r/w operations, you can use _RWMutex_ instead of _Mutex_ to lock r/w operations separately.

----

### channel introduction
After that brief introduction to Go concurrency, we are finally ready to look at a _Channel_ example:
```go
ch := make(chan int)
```
A channel is a high-level data structure used for synchronization and communication between goroutines (see [4]).

Let's perform a simple operation on a channel: we will add 1 into the channel, and get 1 from the channel and print it out:
```go
func main() {
	ch := make(chan int)
	// Writing to the channel will block until something is read from the channel 
	ch <- 1 // blocks here!
	v := <-ch // we get a deadlock :)
	fmt.Println(v)
}
```
result:
```ascii
fatal error: all goroutines are asleep - deadlock!
```
As you can see, we got a deadlock! We got it because operations of writing to a channel and reading from a channel are **blocking**! 
The most important lesson is that channels are made for data exchange and communication **between goroutines**.

Ok, let's fix our code and finally print 1:
```go
func main() {
	ch := make(chan int)
	go func() {
		ch <- 1
	}()
	v := <-ch
	fmt.Println(v)
}
```

----

Note: As you can see, we did not use any _WaitGroup_ for synchronization, because channels already have blocking behavior.

----

**Problems to exercise** (see full list below):
- `concstart`

## Go channel axioms

### These are the axioms of (**unbuffered**) channels:

| Ops   | Open                            | Closed                | Not init (nil channel) |
|-------|---------------------------------|-----------------------|------------------------|
| Read  | **blocked** until writer coming | **return zero value** | deadlock               |
| Write | **blocked** until reader coming | panic                 | deadlock               |
| Close | close channel                   | panic                 | panic                  |

----
Note: additionally, you can mark a channel as read-only (`<- chan int`) or write-only (`chan <- int`). Then, the following additional restrictions are enforced at the compile level:

for **read-only** channels:
- Write - compile error
- Close - compile error

for **write-only** channels:
- Read - compile error
----

(See also the [6] link - it only has 4 idioms, missing some channel states in favor of "more important" ones)

### Examples
Let's go through examples of the (non-buffered) channel axioms.

- write to a nil channel -> deadlock
```go
func main() {
	var ch chan int
	ch <- 1
}
```
result:
```ascii
fatal error: all goroutines are asleep - deadlock!
```
outcome - you must create a channel every time before using it:
```go
ch := make(chan int)
```
or if you use a channel in a struct:
```go
type test struct{
	ch chan int
}
...
t := test{
	ch: make(chan int),
}
```
or if you need to create an array of channels:
```go
out := make([]chan int, numChans)
for i := 0; i < numChans; i++ {
    out[i] = make(chan int, numChans)
}
```

- write to a channel -> deadlock
```go
func main() {
	ch := make(chan int)
	ch <- 1
}
```
result:
```ascii
fatal error: all goroutines are asleep - deadlock!
```
outcome - if the channel does not have any reader, it's blocked until data is read by a reader.
- write to a channel in the main goroutine, and read from it in the same goroutine
```go
func main() {
	ch := make(chan int)
	ch <- 1
	v := <- ch
	fmt.Println(v)
}
```
result:
```ascii
fatal error: all goroutines are asleep - deadlock!
```
outcome - we are still blocked on the `ch <- 1`. We need another goroutine to read the value from the channel. 
**Non-buffered channels are used to exchange data between goroutines.**
- write to a channel in a different goroutine, and read from it in the main goroutine
```go
func main() {
	ch := make(chan int)
	go func() {
		ch <- 1
	}()
	v := <-ch
	fmt.Println(v)
}
```
result:
```ascii
1
```
- write to a channel in a different goroutine, and read from it in the main goroutine twice
```go
func main() {
	ch := make(chan int)
	go func() {
		ch <- 1
	}()
	v := <-ch
	fmt.Println(v)

	v = <-ch
	fmt.Println(v)
}
```
result:
```ascii
1
fatal error: all goroutines are asleep - deadlock!
```
outcome - the second reading from the channel blocks because the value has been read already.
- write to a channel in a different goroutine twice, and read from it in the main goroutine twice
```go
func main() {
	ch := make(chan int)
	go func() {
		ch <- 1
		ch <- 2
	}()
	v := <-ch
	fmt.Println(v)

	v = <-ch
	fmt.Println(v)
}
```
result:
```ascii
1
2
```
outcome - works as expected: two writes, two reads.
- writing an unknown (for the reader) number of elements into a channel in a different goroutine, and reading them correctly in the main goroutine.
In real life, we never know how many elements have been added into a channel before it is returned to us for pulling. However, if a channel is closed 
after writing, we may use the channel axiom that returns the zero value when a channel is closed to stop pulling.
```go
func writer() chan int {
	ch := make(chan int)
	go func() {
		for i := range 5 {
			ch <- i
		}
		close(ch) // close channel after all data has been written
	}()
	return ch
}

func main() {
	ch := writer()
	for { // infinite loop
		v, ok := <-ch
		if !ok {
			break // break loop when the channel is closed
		}
		fmt.Println(v)
	}
}
```
result:
```ascii
0
1
2
3
4
```
In addition, it's good practice to return the channel as a **read-only** channel (see the `<-` before `chan` in the return definition) - `func writer() <-chan int {`.
That will allow the Go compiler to highlight possible errors during the close operation at compile time. Example:
```ascii
Cannot use ch (type <-chan int) as the type chan<- Type
Must be a bidirectional or send-only channel
```
The channel will be converted to a read-only channel only after leaving the function; you can still write values into the channel inside the function. 

## Basic Go channel patterns

### Generator
If a function creates a channel, writes into it, and returns the channel for reading, we call this pattern a **Generator**.
The code between creating the channel and returning it MUST NOT have any blocking operations - all operations must be run in a goroutine.
The following **Generator** uses two goroutines to write into the channel:
```go
func generate() <-chan int {
	ch := make(chan int)
	go func() {
		for i := range 5 {
			ch <- i
		}
		close(ch)
	}()
	go func() {
		for i := range 5 {
			ch <- i + 10
		}
		close(ch)
	}()
	return ch
}

func main() {
	ch := generate()
	for {
		v, ok := <-ch
		if !ok {
			break
		}
		fmt.Println(v)
	}
}
```
result:
```ascii
10
11
0
1
2
3
4
```
outcome: We have an incorrect result because the first goroutine finishes first and closes the channel. 

In addition, we have a race condition that can cause a panic on the closed channel. Let's add a sleep at the end of `main` to wait until the second goroutine tries to write into the closed channel:
```go
func generate() <-chan int {
	ch := make(chan int)
	go func() {
		for i := range 5 {
			ch <- i
		}
		close(ch)
	}()
	go func() {
		for i := range 5 {
			ch <- i + 10
		}
		close(ch)
	}()
	return ch
}

func main() {
	ch := generate()
	for {
		v, ok := <-ch
		if !ok {
			break
		}
		fmt.Println(v)
	}

	time.Sleep(1 * time.Second)
}
```
result:
```ascii
10
11
0
1
2
3
4
panic: send on closed channel
```
Let's modify the "Generator" to use two goroutines to write into the channel, with the correct closing of the channel synchronized with a `WaitGroup`:
```go
func writer() <-chan int {
	ch := make(chan int)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range 5 {
			ch <- i
		}
	}()
	go func() {
		defer wg.Done()
		for i := range 5 {
			ch <- i + 10
		}
	}()

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}

func main() {
	ch := writer()
	for {
		v, ok := <-ch
		if !ok {
			break
		}
		fmt.Println(v)
	}

	time.Sleep(1 * time.Second)
}
```
result:
```ascii
10
11
0
1
2
3
4
12
13
14
```

----

Note: Since, according to the **generator** pattern, we must not have any blocking operations in the function, we wrap up `wg.Wait()`/`close(ch)` into a separate goroutine as well.

----

### range

We do not need to use a loop to pull values and break when the channel is closed:
```go
for {
    v, ok := <-ch
    if !ok {
        break
    }
	...
}
```
We can just use `range` on the channel to pull from it:
```go
for v := range ch {
    fmt.Println(v)
}
```
When using `range`, you MUST close the channel, or you will get a deadlock.

## Select
Let's make a simple _select_ program and run it:
```go
func main() {
	select {}
}
```
result:
```ascii
fatal error: all goroutines are asleep - deadlock!
```
As you can see, **select** is a **blocking** operator.

`select` allows you to read from multiple channels; it acts like a switch for channels (see link [5]).
```go
func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	
	select {
	case v := <-ch1:
		fmt.Println("ch1: ", v)
	case v := <-ch2:
		fmt.Println("ch2: ", v)
	}
}
```
If we run this program now, we will get a deadlock because `select` will iterate through all blocking operations (two in our case), check that they are all blocked, and throw a deadlock (because there is no writer for either channel).
Let's write a value into one of the channels:
```go
func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		ch1 <- 1
	}()
	
	select {
	case v := <-ch1:
		fmt.Println("ch1: ", v)
	case v := <-ch2:
		fmt.Println("ch2: ", v)
	}
}
```
result:
```ascii
ch1:  1

Process finished with the exit code 0
```
The program printed the value and exited, which means `select` **works once** and executes the **first case that becomes unlocked**. 
If the first channel is unlocked first, `select` reads from it and exits.

How to make `select` non-blocking? We can define a `default` case to make it non-blocking:
```go
func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	select {
	case v := <-ch1:
		fmt.Println("ch1: ", v)
	case v := <-ch2:
		fmt.Println("ch2: ", v)
	default:
		fmt.Println("select is non-blocking now")
	}
}
```
result:
```ascii
select is non-blocking now

Process finished with the exit code 0
```

Let's run this code:
```go
func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		ch1 <- 1
	}()
	
	select {
	case v := <-ch1:
		fmt.Println("ch1: ", v)
	case v := <-ch2:
		fmt.Println("ch2: ", v)
	default:
		fmt.Println("select is non-blocking now")
	}
}
```
Would we expect the result to be?
```ascii
ch1:  1

Process finished with the exit code 0
```
However, we actually get:
```ascii
select is non-blocking now

Process finished with the exit code 0
```
Because our goroutine was not scheduled to run before the program exited. To make this program work as expected, we can put a `Sleep` before the `select`.
That will allow the goroutine to run and produce the expected result:
```go
func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		ch1 <- 1
	}()

	time.Sleep(1 * time.Second)

	select {
	case v := <-ch1:
		fmt.Println("ch1: ", v)
	case v := <-ch2:
		fmt.Println("ch2: ", v)
	default:
		fmt.Println("select is non-blocking now")
	}
}
```
```ascii
ch1:  1

Process finished with the exit code 0
```
If we want to exit `select` not by `default` but by timeout, we can use `time.After`:
```go
func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	select {
	case v := <-ch1:
		fmt.Println("ch1: ", v)
	case v := <-ch2:
		fmt.Println("ch2: ", v)
	case <-time.After(1 * time.Second):
		fmt.Println("timeout")
	}
}
```
```ascii
timeout

Process finished with the exit code 0
```
We can also exit using a timer:
```go
func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	timer := time.NewTimer(time.Millisecond * 100)
	select {
	case v := <-ch1:
		fmt.Println("ch1: ", v)
	case v := <-ch2:
		fmt.Println("ch2: ", v)
	case <-time.After(1 * time.Second):
		fmt.Println("timeout")
	case <-timer.C:
		fmt.Println("timer expired")
	}
}
```
```go
timer expired

Process finished with the exit code 0
```
Exit by context:
```go
func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	select {
	case v := <-ch1:
		fmt.Println("ch1: ", v)
	case v := <-ch2:
		fmt.Println("ch2: ", v)
	case <-ctx.Done():
		fmt.Println("timeout")
	}
}
```
```ascii
timeout

Process finished with the exit code 0 
```

How do all three (`after`, `timer`, `context`) work? They use the same pattern: closing a channel when a condition is met. Let's create our own custom `After`:
```go
func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	ch3 := make(chan int)
	go func() {
		ch3 <- 3
		close(ch3)
	}()
	//close(ch3)

	select {
	case v := <-ch1:
		fmt.Println("ch1: ", v)
	case v := <-ch2:
		fmt.Println("ch2: ", v)
	case v := <-ch3:
		fmt.Println("ch3: ", v)
	}
}
```
```go
ch3:  3

Process finished with the exit code 0
```

What is the best way to exit a `select`? **CONTEXT**

Let's consider a simple program with `select`:
```go
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	ch := make(chan int)
	go func() {
		for i := range 10 * 10 * 10 * 10 * 10 {
			ch <- i
		}
		close(ch)
	}()

	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return
			}
			fmt.Println("ch: ", v)
		case <-ctx.Done():
			return
		}
	}
}
```
The program creates a channel, writes N values into it, and awaits processing with a `select` or exits via context cancellation. However, context cancellation itself does not
cancel the goroutine, which creates a **goroutine leak**.
To avoid this, let's also propagate the context cancellation through `select` into the goroutine:
```go
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	ch := make(chan int)
	go func() {
		for i := range 10 * 10 * 10 * 10 * 10 {
			select {
			case ch <- i:
			case <-ctx.Done():
				return
			}
		}
		close(ch)
	}()

	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return
			}
			fmt.Println("ch: ", v)
		case <-ctx.Done():
			return
		}
	}
}
```
You can run both of these code snippets to see the difference.

**Problems to exercise** (see full list below):
- `pipeline`
- `nworkers`
- `longrunner`
- `processdata`
- `processdata_with_context`

## Buffered channels
The Uber Go style guide recommends not using buffered channels (see ref [7] below):
```ascii
Channel Size is One or None

Channels should usually have a size of one or be unbuffered. By default, channels are unbuffered and have a size of zero.
Any other size must be subject to a high level of scrutiny. Consider how the size is determined, what prevents the channel 
from filling up under load and blocking writers, and what happens when this occurs.
```
Bad by Uber style guide:
```go
// Ought to be enough for anybody!
c := make(chan int, 64)
```

Good by Uber style guide:
```go
// Size of one
c := make(chan int, 1) // or
// Unbuffered channel, size of zero
c := make(chan int)
```

A buffered channel works like an unbuffered channel except:
- **WRITE LOCK** on the channel happens only when the **BUFFER** is **FULL**
- **READ LOCK** on the channel happens when the **BUFFER** is **EMPTY**

A buffered channel does not have any OTHER LOCKS, which means we do not have synchronization. A buffered channel does not create a Join point in the same way an unbuffered channel does:
```ascii
        fork->
              \
g0(main) ----> go g1 --------> println ----> exit
                \ 
                g1 --------------------------------> println
```

Let's consider a simple program with an unbuffered channel:
```go
func main() {
	ch := make(chan int)
	go func() {
		// read
		for range 3 {
			v := <-ch
			fmt.Println(v)
		}
		close(ch)
	}()
    // write
	ch <- 1
	ch <- 2
	ch <- 3
}
```
result:
```ascii
1
2
3

Process finished with the exit code 0
```

Now let's change the unbuffered channel to a buffered one with a size of 3:
```go
func main() {
	ch := make(chan int, 3)
	go func() {
		// read
		for range 3 {
			v := <-ch
			fmt.Println(v)
		}
	}()
    // write
	ch <- 1
	ch <- 2
	ch <- 3
}
```
result:
```ascii

Process finished with the exit code 0
```
As you can see, synchronization is no longer working as it does with unbuffered channels. Since the writer had enough slots to fill up the buffer, it was not blocked and the program exited before the scheduler could run the reading goroutine.
If we add a sleep at the end of the program or increase the buffer size and writes, the program will read all or part of the data:
```go
func main() {
	ch := make(chan int, 10000)
	go func() {
		for range 3 {
			v := <-ch
			fmt.Println(v)
		}
	}()

	for i := range 10000 {
		ch <- i
	}
}
```
```ascii
0
1
2
3
4
...
```

Why would we need a buffered channel? 

**Buffered channels are non-blocking for the sender as long as there's still room. This can increase responsiveness and throughput. 
Sending several items on a buffered channel ensures they are processed in the order in which they were sent.**
Example:
```go
func main() {
	ch := make(chan int, 3)

	ch <- 1
	fmt.Println("v 1 is written")
	ch <- 2
	fmt.Println("v 2 is written")
	ch <- 3
	fmt.Println("v 3 is written")

	close(ch)

	go func() {
		time.Sleep(1 * time.Second)
		for v := range ch {
			fmt.Println("v ", v)
		}
	}()

	time.Sleep(5 * time.Second)
}
```

## Goroutine leaks
A goroutine leak happens when the result of a goroutine's work is no longer needed, but the goroutine keeps working and consuming memory, CPU, and other resources. 

Let's look at a simple example of a goroutine leak:
```go
func leakyG() {
	go func(){
	    for {}	
    }
}
```
The goroutine in this example will never finish.

Let's look at an example with a "forgotten" reader/writer.
Initial example:
```go
func main() {
	ch := make(chan int)

	// writer
	go func() {
		defer fmt.Println("writer is done")
		for i := range 10 {
			ch <- i
		}
	}()

	// reader
	go func() {
		defer fmt.Println("reader is done")
		for v := range ch {
			fmt.Println("read", v)
		}
	}()

	time.Sleep(1 * time.Second)
}
```
result:
```ascii
read 0
read 1
writer is done
read 2
```
The writer has exited, but the reader has not because we did not close the channel after writing.
Let's fix it:
```go
func main() {
	ch := make(chan int)

	// writer
	go func() {
		defer fmt.Println("writer is done")
		for i := range 3 {
			ch <- i
		}
		close(ch)
	}()

	// reader
	go func() {
		defer fmt.Println("reader is done")
		for v := range ch {
			fmt.Println("read", v)
		}
	}()

	time.Sleep(1 * time.Second)
}
```
result:
```ascii
read 0
read 1
writer is done
read 2
reader is done
```
Great, now both have exited.

Now let's emulate an issue in the reader goroutine:
```go
func main() {
	ch := make(chan int)

	// writer
	go func() {
		defer fmt.Println("writer is done")
		for i := range 3 {
			ch <- i // oh no, writer is blocked now
		}
		close(ch)
	}()

	// reader
	go func() {
		defer fmt.Println("reader is done")
		for v := range ch {
			fmt.Println("read", v)
			return // some issue on the reader
		}
	}()

	time.Sleep(1 * time.Second)
}
```
result:
```ascii 
read 0
reader is done
```
The writer goroutine is now leaked. To avoid this, we can add a cancellation context as we did before:
```go
func main() {
	ch := make(chan int)

	ctx, cancel := context.WithCancel(context.Background())

	// writer
	go func() {
		defer fmt.Println("writer is done")
		for i := range 3 {
			select {
			case ch <- i:
			case <-ctx.Done():
				return
			}
		}
		close(ch)
	}()

	// reader
	go func() {
		defer fmt.Println("reader is done")
		for v := range ch {
			fmt.Println("read", v)

			cancel()
			return // some issue on the reader
		}
	}()

	time.Sleep(1 * time.Second)
}
```
result:
```go
read 0
reader is done
writer is done
```
Nice, both goroutines have exited as expected. But what if we get an error in the writer?
```go
func main() {
	ch := make(chan int)

	ctx, _ := context.WithCancel(context.Background())

	// writer
	go func() {
		defer fmt.Println("writer is done")
		for i := range 3 {
			select {
			case ch <- i:
				return // some issue on the writer
			case <-ctx.Done():
				return
			}
		}
		close(ch)
	}()

	// reader
	go func() {
		defer fmt.Println("reader is done")
		for v := range ch {
			fmt.Println("read", v)
		}
	}()

	time.Sleep(1 * time.Second)
}
```
result:
```ascii 
writer is done
read 0
```
Now there's a leak in the reader! That's because our channel was not closed yet - we only close it at the end of the writer goroutine.
If we use `defer` for closing, we can avoid this problem:
```go
func main() {
	ch := make(chan int)

	ctx, _ := context.WithCancel(context.Background())

	// writer
	go func() {
		defer close(ch)
		defer fmt.Println("writer is done")
		for i := range 3 {
			select {
			case ch <- i:
				return // some issue on the writer
			case <-ctx.Done():
				return
			}
		}
		//close(ch)
	}()

	// reader
	go func() {
		defer fmt.Println("reader is done")
		for v := range ch {
			fmt.Println("read", v)
		}
	}()

	time.Sleep(1 * time.Second)
}
```
result:
```ascii
writer is done
read 0
reader is done
```

----

Note: Use **defer** to close channels whenever possible.

----

**Problems to exercise**:
- `fanin_fanout_workerpool`

## Problems to exercise
- _Easy_ [Go concurrency bare minimum] `concstart` - You have a function running between 0 and N seconds. Run this function concurrently M times
and print out how many seconds the main function runs and how many seconds all functions run in parallel.
    - Solution 1 (with `WaitGroup`) - `./concstart/wg/main.go`
    - Solution 2 (with `channel`) - `./concstart/ch/main.go`
- _Medium_ [Go channel axioms] `pipeline` - Create 3 functions: _writer_ - generates numbers from 0 to 20; _doubler_ - multiplies numbers by 2 with a sleep of 500ms simulating some IO-bound work;
_reader_ - reads and prints out on the screen. All functions must be implemented in a concurrent way and synchronized accordingly.
    - Solution: `./pipeline/main.go`
- _Easy_ [Go channel axioms] `nworkers` - Create a channel and create M goroutines (2,3,4...) writing into it, then create N different goroutines (2,3,4...) reading from the channel.
    - Solution: `./nworkers/main.go`
- _Easy_ [Go channel axioms] `longrunner` - You have a function with undefined behavior - it can take between 1 and 100 seconds. Create a wrapper for this function
to break execution if it takes more than 3 seconds and return an error.
    - Solution 1 (with `time.After`): `./longrunner/after/main.go`
    - Solution 2 (with context): `./longrunner/context/main.go`
- _Medium_ [Go channel axioms] `processdata` - You have a function to process data which, for simplicity, takes an integer K and returns K * 2 after some wait (let's say it waits between 0 and 10 seconds).
 Write data (10 numbers as an example) into a buffer concurrently and then process the data in parallel with N workers.
    - Solution: `./processdata/main.go`
- _Hard_ [Go channel axioms] `processdata_with_context` - You have a function to process data which, for simplicity, takes an integer K and returns K * 2 after some wait (let's say it waits between 0 and 10 seconds). 
Write data (10 numbers as an example) into a buffer concurrently and then process the data in parallel with N workers. Each process should run no longer than M seconds (5 by example) and print out the execution time.
Please implement cancellation using a timeout context.
    - Solution: `./processdata_with_context/main.go`
- _Medium_ [Advanced channel patterns] _tee pattern_ - Implement the "tee" pattern - a function that has one input channel and N output channels (identical to the input). Please implement cancellation using a timeout context.
  - Solution: `./tee/main.go`
- _Hard**_ [Advanced channel patterns] `fanin_fanout_workerpool` - Implement fan-in / fan-out and a worker pool, both with context cancellation. Using metrics, show the advantage of one over the other, if such an advantage exists.
    - Solution: `./fanin_fanout_workerpool/cmd/main.go`

## References
- [1] Abandoned but still beautiful blog of Dmitry Vyukov - https://sites.google.com/site/1024cores/home
- [2] Go Scheduler concepts by Dmitry Vyukov on Hydra conf - https://www.youtube.com/watch?v=-K11rY57K7k 
- [3] Go `WaitGroup` - https://github.com/golang/go/blob/master/src/sync/waitgroup.go
- [4] Go `channel` - https://go.dev/src/runtime/chan.go
- [5] Go `select` - https://go.dev/src/runtime/select.go
- [6] Go channel axioms - https://dave.cheney.net/2014/03/19/channel-axioms
- [7] Uber Go style guide - buffered vs unbuffered channels - https://github.com/uber-go/guide/blob/master/style.md#channel-size-is-one-or-none
