# Channels

## What are Channels?

Channels are Go's way of allowing goroutines to communicate with each other safely. They're like pipes that connect goroutines - one goroutine can send data into the channel, and another can receive it.

**Go Philosophy**: "Don't communicate by sharing memory; share memory by communicating."

## Basic Syntax

### Creating Channels

```go
// Create a channel of integers
ch := make(chan int)

// Create a buffered channel (holds up to 5 values)
ch := make(chan int, 5)

// Create a channel of strings
messages := make(chan string)
```

### Sending and Receiving

```go
ch := make(chan int)

// Send value into channel
ch <- 42

// Receive value from channel
value := <-ch

// Receive and ignore value
<-ch
```

## Simple Example

```go
package main

import "fmt"

func main() {
    // Create a channel
    messages := make(chan string)

    // Send a message (in a goroutine, so it doesn't block)
    go func() {
        messages <- "Hello from goroutine!"
    }()

    // Receive the message
    msg := <-messages
    fmt.Println(msg)  // Output: Hello from goroutine!
}
```

## Unbuffered vs Buffered Channels

### Unbuffered Channels (Default)

**Synchronous** - sender blocks until receiver is ready:

```go
ch := make(chan int)  // Unbuffered

go func() {
    ch <- 1  // Blocks until someone receives
}()

value := <-ch  // Blocks until someone sends
```

**Use case:** When you need synchronization between goroutines.

### Buffered Channels

**Asynchronous** - sender only blocks when buffer is full:

```go
ch := make(chan int, 3)  // Buffer size 3

ch <- 1  // Doesn't block
ch <- 2  // Doesn't block
ch <- 3  // Doesn't block
ch <- 4  // BLOCKS - buffer is full!

<-ch  // Receive - now sender can continue
```

**Use case:** When you want to decouple sender/receiver timing.

## Channel Directions

You can specify if a channel is send-only or receive-only:

```go
// Send-only channel
func sendOnly(ch chan<- int) {
    ch <- 42  // OK
    // value := <-ch  // Compile error!
}

// Receive-only channel
func receiveOnly(ch <-chan int) {
    value := <-ch  // OK
    // ch <- 42  // Compile error!
}

func main() {
    ch := make(chan int)

    go sendOnly(ch)
    receiveOnly(ch)
}
```

**Why?** Prevents mistakes and makes code clearer about intent.

## Closing Channels

Senders can close channels to indicate no more values will be sent:

```go
ch := make(chan int, 3)

// Send some values
ch <- 1
ch <- 2
ch <- 3

// Close the channel
close(ch)

// Receivers can detect closure
value, ok := <-ch
if ok {
    fmt.Println("Received:", value)
} else {
    fmt.Println("Channel closed!")
}
```

**Important Rules:**
1. Only the **sender** should close a channel (not the receiver)
2. Sending on a closed channel causes a **panic**
3. Receiving from a closed channel returns the **zero value** immediately
4. Closing a `nil` channel causes a **panic**
5. Closing a channel twice causes a **panic**

## Range Over Channels

You can use `range` to receive values until the channel is closed:

```go
ch := make(chan int, 5)

// Send some values and close
go func() {
    for i := 1; i <= 5; i++ {
        ch <- i
    }
    close(ch)  // Important: close when done
}()

// Receive all values until closed
for value := range ch {
    fmt.Println(value)
}
// Loop exits when channel is closed
```

## Select Statement

`select` lets you wait on multiple channels simultaneously:

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    ch1 := make(chan string)
    ch2 := make(chan string)

    go func() {
        time.Sleep(1 * time.Second)
        ch1 <- "one"
    }()

    go func() {
        time.Sleep(2 * time.Second)
        ch2 <- "two"
    }()

    // Wait for both messages
    for i := 0; i < 2; i++ {
        select {
        case msg1 := <-ch1:
            fmt.Println("Received from ch1:", msg1)
        case msg2 := <-ch2:
            fmt.Println("Received from ch2:", msg2)
        }
    }
}
```

### Select with Default (Non-blocking)

```go
select {
case msg := <-ch:
    fmt.Println("Received:", msg)
default:
    fmt.Println("No message available")
}
```

### Select with Timeout

```go
select {
case result := <-ch:
    fmt.Println("Got result:", result)
case <-time.After(5 * time.Second):
    fmt.Println("Timeout - no result after 5 seconds")
}
```

## Common Patterns

### 1. Worker Pattern

```go
func worker(id int, jobs <-chan int, results chan<- int) {
    for job := range jobs {
        fmt.Printf("Worker %d processing job %d\n", id, job)
        results <- job * 2  // Process and send result
    }
}

func main() {
    jobs := make(chan int, 100)
    results := make(chan int, 100)

    // Start 3 workers
    for w := 1; w <= 3; w++ {
        go worker(w, jobs, results)
    }

    // Send 5 jobs
    for j := 1; j <= 5; j++ {
        jobs <- j
    }
    close(jobs)

    // Collect results
    for r := 1; r <= 5; r++ {
        <-results
    }
}
```

### 2. Pipeline Pattern

```go
// Stage 1: Generate numbers
func generate(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        for _, n := range nums {
            out <- n
        }
        close(out)
    }()
    return out
}

// Stage 2: Square numbers
func square(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        for n := range in {
            out <- n * n
        }
        close(out)
    }()
    return out
}

func main() {
    // Pipeline: generate -> square
    numbers := generate(1, 2, 3, 4)
    squares := square(numbers)

    // Print results
    for result := range squares {
        fmt.Println(result)  // 1, 4, 9, 16
    }
}
```

### 3. Fan-Out, Fan-In Pattern

```go
// Fan-out: distribute work to multiple workers
// Fan-in: collect results from multiple workers

func fanIn(channels ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup

    // Start a goroutine for each input channel
    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for n := range c {
                out <- n
            }
        }(ch)
    }

    // Close output when all inputs are done
    go func() {
        wg.Wait()
        close(out)
    }()

    return out
}
```

### 4. Done Channel Pattern

Signal completion or cancellation:

```go
func worker(done <-chan bool) {
    for {
        select {
        case <-done:
            fmt.Println("Worker stopping")
            return
        default:
            // Do work
            time.Sleep(100 * time.Millisecond)
        }
    }
}

func main() {
    done := make(chan bool)

    go worker(done)

    time.Sleep(1 * time.Second)
    done <- true  // Signal worker to stop
    time.Sleep(100 * time.Millisecond)
}
```

## Using Channels in Connversed

### Example: Message Broadcasting with Channels

Instead of directly calling `client.Write()`, you could use channels:

```go
type Room struct {
    Id       string
    Name     string
    Clients  []*Client
    Messages chan Message  // Message queue
}

type Message struct {
    Sender  *Client
    Content string
}

// Start message broadcaster
func (r *Room) StartBroadcaster() {
    go func() {
        for msg := range r.Messages {
            for _, client := range r.Clients {
                client.Write(msg.Content, msg.Sender)
            }
        }
    }()
}

// Send message to queue
func (r *Room) SendMessage(sender *Client, content string) {
    r.Messages <- Message{Sender: sender, Content: content}
}
```

**Benefits:**
- Decouples message sending from broadcasting
- Messages are processed in order
- Can add features like rate limiting, filtering, etc.

## Common Pitfalls

### 1. Deadlock - Sending Without Receiver

```go
// Wrong - deadlock!
ch := make(chan int)
ch <- 1  // Blocks forever - no receiver

// Right
ch := make(chan int)
go func() {
    ch <- 1
}()
value := <-ch
```

### 2. Deadlock - Waiting on Empty Channel

```go
// Wrong - deadlock!
ch := make(chan int)
value := <-ch  // Blocks forever - no sender

// Right
ch := make(chan int)
go func() {
    ch <- 1
}()
value := <-ch
```

### 3. Sending on Closed Channel

```go
// Wrong - panic!
ch := make(chan int)
close(ch)
ch <- 1  // panic: send on closed channel

// Right - check if you should close
ch := make(chan int)
// ... use channel ...
close(ch)
// Don't send after closing
```

### 4. Not Closing Channels in Range

```go
// Wrong - range waits forever
ch := make(chan int)
go func() {
    ch <- 1
    ch <- 2
    // Forgot to close!
}()

for value := range ch {
    fmt.Println(value)  // Hangs after 1, 2
}

// Right
ch := make(chan int)
go func() {
    ch <- 1
    ch <- 2
    close(ch)  // Signal done
}()

for value := range ch {
    fmt.Println(value)  // Prints 1, 2, then exits
}
```

## Best Practices

1. **Only sender should close** - receivers shouldn't close channels
2. **Don't close if not needed** - garbage collection handles unclosed channels
3. **Use buffered channels** for asynchronous operations
4. **Use unbuffered channels** for synchronization
5. **Check `ok` value** when closure matters: `value, ok := <-ch`
6. **Use `select` with `default`** for non-blocking operations
7. **Prefer channels over shared memory** for communication

## See Also

- [Goroutines](goroutines.md) - Concurrent execution
- [WaitGroup](waitgroup.md) - Waiting for goroutines
- [Race Conditions](race-conditions.md) - Thread safety
- [Concurrency Patterns](concurrency-patterns.md) - Advanced patterns
