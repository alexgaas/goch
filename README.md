# Go Channels cheatsheet
This is really simple cheatsheet to understand how to use correctly channels with goroutines in a Go. This basic introduction does not assume 
consideration of Go channels under the hood and instead of that focuses on the practical aspects of using channels on to production-ready services.

## Go concurrency bare minimum
Let's consider a simple example of program with goroutine:
```go
func main() {
	go func() { // g1
		fmt.Println("test")
	}()
	fmt.Println("Hello World")
}
```
If we run this program we would never see goroutine is started b/c exit from program happen before goroutine is really joined to _g0_: 

```ascii
        fork->
              \
g0(main) ----> go g1 ----> println ----> exit
                \
                g1 -----------------------------> println
```
That happens b/c we do not control when goroutine start in the program - Go scheduler does it for us (some details about how scheduler works you can find looking into ref [2]).

----
Note: If we want any concurrent program on Go to run correctly, we have to:
- synchronize goroutines
- prevent race / contention
- address any possible goroutine leaks (will be considered in different topic)
----

### synchronize goroutines

Model of Go concurrency is based off **Fork/Join** approach (it is also called parallel divide and conquer - see ref [1] below). To address problem we had before and start our goroutine we need to add **Join** point into our program - e.g. synchronize goroutines. 
```ascii
        fork->
              \
g0(main) ----> go g1 --------> println ----> exit
                \ -> join ^ 
                g1 ----> println
```
How would we do that sync (add join point) in the Go? We can add a **wait group**:
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

result of this execution will be:
```ascii
test
Hello World
```

----

Note: please make you pass `WaitGroup` into function by reference. If you pass by value, wait group will be copied and never wait end of goroutine!

----

### prevent race / contention

Let's consider a contention problem when N goroutines trying to get and modify one variable in the memory.
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
This program tries to increment one variable with N goroutines (1000 in our case). If we run program we'd get following result:
```ascii
927
...
898
...
946
```
but not a 1000 as expect. The reason of that `m = m + 1` is not atomic operation, it consists of three operations: read m -> add one -> write m. 

flow diagram:
```ascii
main g0 \ -> m = 0
        go g1() -> r/m/w = 0/1/1
        go g2() -> r/m/w = 0/1/1
        /
m = 1, m != 2 as expected
```
That is a classic case of data contention (or race). How would we address this problem? Most simple way would be to use **atomic** types:
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
Another option would be to use _Mutex_ - a mutual exclusion lock.
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
Advantage of mutex is obvious - in given moment of time only one goroutine may do set of instructions for you (but not only certain atomic type though!).

----

Note: if you have a few goroutines with separate r/w operations we can use _RWMutex_ instead of _Mutex_ to lock r/w ops separately.

----

### channel introduction
After that really brief introduction to Go concurrency we finally ready to look on the _Channel_ example:
```go
ch := make(chan int)
```
Channel is a high level data structure used as for synchronization and communications between goroutines (see [4]).

Let's make a simple operation on the channel, will add into channel 1, and get from the channel 1 and print it out:
```go
func main() {
	ch := make(chan int)
	// Writing to channel won't be unlocked until anything will be read from channel 
	ch <- 1 // since locked! here already 
	v := <-ch // we get deadlock :)
	fmt.Println(v)
}
```
result:
```ascii
fatal error: all goroutines are asleep - deadlock!
```
As you may see we got deadlock! We got it b/c operations of writing to channel and reading from channel are **blocking**! 
Most important lesson of that must - channels made for data exchange/communications **between goroutines**.

Ok let's fix our code and print out 1 finally:
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

Note_: As you may see we did not use any _WaitGroup_ for synchronization, b/c channel already having blocking behavior

----

**Problems to exercise** (see full list below):
- `concstart`

## Go channel axioms

### There are axioms of (**unbuffered**) channel:

| Ops   | Open                            | Closed                | Not init (nil channel) |
|-------|---------------------------------|-----------------------|------------------------|
| Read  | **blocked** until writer coming | **return zero value** | deadlock               |
| Write | **blocked** until reader coming | panic                 | deadlock               |
| Close | close channel                   | panic                 | panic                  |

----
Note: additionally you can mark channel as read-only (`<- chan int`) or write-only (`chan <- int`), then following additional restrictions happen on the compile level:

for **read-only** channels:
- Read - compile error
- Write - compile error

for **write-only** channels:
- Read - compile error
----

(also see [6] link - it though has only 4 idioms missing some channel states in favor of "more important")

### Examples
Let's go through examples on the (non-buffered) channel axioms.

- write to nil channel -> deadlock
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
as outcome - you must create channel every time before use it:
```go
ch := make(chan int)
```
or if you use channel in the structure:
```go
type test struct{
	ch chan int
}
...
t := test{
	ch: make(chan int),
}
```
or if you need to create array of channels:
```go
out := make([]chan int, numChans)
for i := 0; i < numChans; i++ {
    out[i] = make(chan int, numChans)
}
```

- write to channel -> deadlock
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
outcome - (channel does not have any reader) it's blocked until data read by any reader
- write to channel in main goroutine, read from channel in same goroutine
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
outcome - we still blocked on the `ch <- 1`, we need any channel to read value from **different** goroutine. 
**Non-buffered channels need to exchange data between goroutines.**
- write to channel in different goroutine, read from channel in main goroutine
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
- write to channel in different goroutine, read from channel in main goroutine twice
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
outcome - second reading from channel blocks it b/c value have been read already
- write to channel in different goroutine twice, read from channel in main goroutine twice
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
outcome - works as expected, two times write, two times read
- write (unknown for reader) number of elements into channel in different goroutine, read them correctly in main goroutine.
In real life we never know how many elements have been added into channel before it's been returned to us for pulling. However, if channel is closed 
after writing we may use channel axiom to return zero when channel is closed to stop pulling.
```go
func writer() chan int {
	ch := make(chan int)
	go func() {
		for i := range 5 {
			ch <- i
		}
		close(ch) // close channel after all data have been written
	}()
	return ch
}

func main() {
	ch := writer()
	for { // infinite loop
		v, ok := <-ch
		if !ok {
			break // break loop when got 0 from closed channel
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
In addition, as a good practice would be make return channel as **read-only** channel (see <- before chan definition on return) - `func writer() <-chan int {`.
That will give context to go compiler to highlight possible errors on close operation in compile time. Example:
```ascii
Cannot use ch (type <-chan int) as the type chan<- Type
Must be a bidirectional or send-only channel
```
Channel will be converted to read-only channel only after leaving function, you still can write values into channel inside of function. 

## Basic Go channel patterns

### Generator
If function creates channel / writes into channel and returns after channel for reading we call this pattern **Generator**.
Function between creating of channel and returning it out MUST NOT have any blocking operation - all operations must be run in goroutine.
**Generator** uses two goroutines to write into channel
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
outcome: we have incorrect result b/c first goroutine finishes first and closes channel. 

In addition to we have a race condition what can cause panic on the closed channel. Let's make a sleep in the end of main to wait until second goroutine will try to write into closed channel:
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
Let's make "Generator" to use two goroutines to write into channel with correct closing of channel synced with `WaitGroup`:
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

Note: since in according to **generator** pattern, we must not have any blocking operations in the function, we wrap up `wg.Wait()`/`close(ch)` into separate goroutine as well.

----

### range

We do not need to use construct to pull values and break on zero when channel is closed:
```go
for {
    v, ok := <-ch
    if !ok {
        break
    }
	...
}
```
We can just use a `range` on the channel to pull from it as:
```go
for v := range ch {
    fmt.Println(v)
}
```
In case of using range you MUST close a channel, or you will get deadlock.

## Select
Let's make a simplest _select_ program and run it:
```go
func main() {
	select {}
}
```
result:
```ascii
fatal error: all goroutines are asleep - deadlock!
```
As you may see - **select** is a **blocking** operator.

Select allows you to read from either one channel - using select you can make a switch between channels (see link [5]).
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
If we run this program now we will get deadlock b/c `select`will iterate through all blocking operations (two in our case) and check they all blocked and throws deadlock (b/c no any writer to corresponding channel).
Let's write into one channel any value:
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
so we printed value and exited program what means - `select` **works one time** and get **case first which was unlocked first**. 
If we unlocked first channel 1, `select` reads from channel 1 and exited select.

How to make `select` non-blocking? We can define `default` on the select to make it non-blocking:
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
Do we assume to get result as correct?
```ascii
ch1:  1

Process finished with the exit code 0
```
However, we're really getting again:
```ascii
select is non-blocking now

Process finished with the exit code 0
```
B/c our goroutine was not put scheduler for a run before program is exited. To make this program working as expected we can just put Sleep in front of `select`.
That will goroutine and get result as we expect:
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
If we want to leave `select` not by `default` but by timeout we can use `time.After` in the case:
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
We also can leave by timer:
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
Leave by context:
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

How all three `after`/`timer`/`context` work? They use a same pattern to close channel after meeting some condition. Let's make our custom `After`:
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

What the best way to leave `select`? **CONTEXT**

Let's consider a simple program  with `select`:
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
program creates channel, writes N values into it and await processing with a `select` or exit by context cancellation. However context cancellation itself does not
cancel goroutine what create **goroutine leak**.
To avoid let's also propagate context cancellation through `select` into goroutine.
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
you can run both code snippets to see the difference.

**Problems to exercise** (see full list below):
- `pipeline`
- `nworkers`
- `longrunner`
- `processdata`
- `processdata_with_context`

## Buffered channels
Uber Go style guide dictates to not use buffered channels (see ref [7] below):
```ascii
Channel Size is One or None

Channels should usually have a size of one or be unbuffered. By default, channels are unbuffered and have a size of zero.
Any other size must be subject to a high level of scrutiny. Consider how the size is determined, what prevents the channel 
from filling up under load and blocking writers, and what happens when this occurs.
```
Bad	by Uber style guide:
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

Buffered channel works as unbuffered channel except:
- **WRITE LOCK** on the channel happens only when **BUFFER** is **FULLY FILLED**
- **READ LOCK** on the channel happens when **BUFFER** is **EMPTY**

Buffered channel does not have any OTHER LOCKS what means we do not have synchronization. Buffered channel does not make a Join point as it does unbuffered channel:
```ascii
        fork->
              \
g0(main) ----> go g1 --------> println ----> exit
                \ 
                g1 --------------------------------> println
```

Let's make a simple program with unbuffered channel:
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
```ascii1
2
3

Process finished with the exit code 0
```

Now let's change unbuffered to buffered with a size 3:
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
As you may see synchronization is no longer working expected for unbuffered channel way. Since writer had enough slots to fill up buffer it's not blocked itself and exited program before scheduler put reading goroutine to work.
If we add sleep in the end of program or increase number of buffer and writes into buffer program will read all or part of data:
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
Sending several items on one buffered channel makes sure they are processed in the order in which they are sent.**
As example:
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
Goroutine leak happens when result of goroutine work no longer need but goroutine keeps working and consume memory and CPU and other resources. 

Let's make a simple example of goroutine leak:
```go
func leakyG() {
	go func(){
	    for {}	
    }
}
```
The goroutine from example would never be finished.

Let's look on the example with "forgotten" reader / writer.
Here is initial example:
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
run and:
```ascii
read 0
read 1
writer is done
read 2
```
Writer is exited but reader is not b/c we did not close channel after writing.
Let's address it:
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
and run:
```ascii
read 0
read 1
writer is done
read 2
reader is done
```
Great, now both are exited.

Now let's emulate issue on the reader goroutine:
```go
func main() {
	ch := make(chan int)

	// writer
	go func() {
		defer fmt.Println("writer is done")
		for i := range 3 {
			ch <- i // oh, no writer is blocked now
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
Writer goroutine is leaked now. To avoid we can add cancellation context as we did before to prevent this:
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
Nice, both goroutines are exited as we expect now. but what if we get a error on the writer now?
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
Now leak on the reader! that's b/c our was not closed yet - we close on the end of writer goroutine.
If we do closing with _defer_ we can avoid this problem
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

Note: use **defer** all time to close channel if it is possible.

----

**Problems to exercise**:
- `fanin_fanout_workerpool`

## Problems to exercise
- _Easy_ [Go concurrency bare minimum] `concstart` - You have a function running between 0 and N seconds. Run this function concurrently M times
and print out how many seconds runs main and how many seconds run all functions in parallel.
    - Solution 1 (with `WaitGroup`) - `./concstart/wg/main.go`
    - Solution 2 (with `channel`) - `./concstart/ch/main.go`
- _Medium_ [Go channel axioms] `pipeline` - Make 3 functions: _writer_ - generates numbers from 0 to 20, _doubler_ - multiplies numbers on 2 with sleep of 500ms simulating some IO-bound work,
_reader_ - reads and prints out on the screen. All functions must be implemented concurrent way and synced accordingly.
    - Solution: `./pipeline/main.go`
- _Easy_ [Go channel axioms] `nworkers` - Create channel and create M goroutine (2,3,4...) writing into channel, then create N different goroutines (2,3,4...) reading from the channel.
    - Solution: `./nworkers/main.go`
- _Easy_ [Go channel axioms] `longrunner` - You have a function with undefined behavior - it can work in the range between 1 and 100 seconds. make a wrapper for this function
to break execution if it takes more than 3 seconds and return error.
    - Solution 1 (with `time.After`): `./longrunner/after/main.go`
    - Solution 2 (with context): `./longrunner/context/main.go`
- _Medium_ [Go channel axioms] `processdata` - You have a function to process data which for simplicity takes an integer number K and returns K * 2 after some wait (let's say it awaits between 0 and 10 seconds).
 Write data (10 numbers as example) into some buffer concurrently and then process data in parallel with N number of workers.
    - Solution: `./processdata/main.go`
- _Hard_ [Go channel axioms] `processdata_with_context` - You have a function to process data which for simplicity takes an integer number K and returns K * 2 after some wait (let's say it awaits between 0 and 10 seconds). 
write data (10 numbers as example) into some buffer concurrently and then process data in parallel with N number of workers, every process should run no longer than M seconds (5 by example) and print out time of execution.
Please implement possible cancellation with timeout context.
    - Solution: `./processdata_with_context/main.go`
- _Medium_ [Advanced channel patterns] _tee pattern_ - Implement "tee" pattern - function what has one channel as input and N (same as an input one) channels as output. Please implement possible cancellation with timeout context.
  - Solution: `./tee/main.go`
- _Hard**_ [Advanced channel patterns] `fanin_fanout_workerpool` - Implement fan-in / fan-out and work pool both with context cancellation. Using metrics show advantage one above other if such as advantage is exist.
    - Solution: `./fanin_fanout_workerpool/cmd/main.go`

## References
- [1] Abandoned but still beautiful blog of Dmitry Vyukov - https://sites.google.com/site/1024cores/home
- [2] Go Scheduler concepts by Dmitry Vyukov on Hydra conf - https://www.youtube.com/watch?v=-K11rY57K7k 
- [3] Go `WaitGroup` - https://github.com/golang/go/blob/master/src/sync/waitgroup.go
- [4] Go `channel` - https://go.dev/src/runtime/chan.go
- [5] Go `select` - https://go.dev/src/runtime/select.go
- [6] Go channel axioms - https://dave.cheney.net/2014/03/19/channel-axioms
- [7] Uber Go style guide - buffered vs unbuffered channels - https://github.com/uber-go/guide/blob/master/style.md#channel-size-is-one-or-none
