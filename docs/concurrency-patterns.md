# Go Concurrency Patterns

This document covers common concurrency patterns in Go that are useful for building robust concurrent applications.

## Table of Contents

1. [Worker Pool](#worker-pool)
2. [Pipeline](#pipeline)
3. [Fan-Out, Fan-In](#fan-out-fan-in)
4. [Timeout Pattern](#timeout-pattern)
5. [Context for Cancellation](#context-for-cancellation)
6. [Done Channel](#done-channel)
7. [Rate Limiting](#rate-limiting)
8. [Semaphore](#semaphore)
9. [Pub/Sub (Broadcast)](#pubsub-broadcast)

---

## Worker Pool

Distribute work across a fixed number of workers to limit concurrency.

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

func worker(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup) {
    defer wg.Done()

    for job := range jobs {
        fmt.Printf("Worker %d started job %d\n", id, job)
        time.Sleep(time.Second)  // Simulate work
        fmt.Printf("Worker %d finished job %d\n", id, job)
        results <- job * 2
    }
}

func main() {
    const numWorkers = 3
    const numJobs = 10

    jobs := make(chan int, numJobs)
    results := make(chan int, numJobs)
    var wg sync.WaitGroup

    // Start workers
    for w := 1; w <= numWorkers; w++ {
        wg.Add(1)
        go worker(w, jobs, results, &wg)
    }

    // Send jobs
    for j := 1; j <= numJobs; j++ {
        jobs <- j
    }
    close(jobs)

    // Wait for workers
    wg.Wait()
    close(results)

    // Collect results
    for result := range results {
        fmt.Println("Result:", result)
    }
}
```

**Use cases:**
- Processing many tasks with limited resources
- Download/upload pools
- Database connection pools

**Benefits:**
- Controls resource usage
- Prevents overwhelming system
- Predictable concurrency level

---

## Pipeline

Chain multiple stages where output of one stage is input to the next.

```go
package main

import "fmt"

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

// Stage 3: Filter even numbers
func filterEven(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        for n := range in {
            if n%2 == 0 {
                out <- n
            }
        }
        close(out)
    }()
    return out
}

func main() {
    // Pipeline: generate -> square -> filterEven
    numbers := generate(1, 2, 3, 4, 5)
    squared := square(numbers)
    evens := filterEven(squared)

    // Print results
    for result := range evens {
        fmt.Println(result)  // 4, 16
    }
}
```

**Use cases:**
- Data processing pipelines
- Stream processing
- ETL (Extract, Transform, Load)

**Benefits:**
- Each stage runs concurrently
- Clear separation of concerns
- Easy to add/remove stages

---

## Fan-Out, Fan-In

**Fan-out**: Distribute work to multiple goroutines
**Fan-in**: Collect results from multiple goroutines

```go
package main

import (
    "fmt"
    "sync"
)

// Generate numbers (1 producer)
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

// Square numbers (fan-out: multiple workers)
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

// Fan-in: merge multiple channels into one
func merge(channels ...<-chan int) <-chan int {
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

func main() {
    // Generate numbers
    numbers := generate(1, 2, 3, 4, 5, 6, 7, 8)

    // Fan-out: 3 workers process in parallel
    c1 := square(numbers)
    c2 := square(numbers)
    c3 := square(numbers)

    // Fan-in: merge results
    results := merge(c1, c2, c3)

    // Print results
    for result := range results {
        fmt.Println(result)
    }
}
```

**Use cases:**
- Parallel processing
- Load balancing
- Aggregating results from multiple sources

---

## Timeout Pattern

Add timeouts to operations using `select` and `time.After`.

```go
package main

import (
    "fmt"
    "time"
)

func doWork() <-chan string {
    result := make(chan string)
    go func() {
        time.Sleep(2 * time.Second)  // Simulate slow work
        result <- "Work complete"
    }()
    return result
}

func main() {
    result := doWork()

    select {
    case res := <-result:
        fmt.Println("Success:", res)
    case <-time.After(1 * time.Second):
        fmt.Println("Timeout: work took too long")
    }
}
```

**Use cases:**
- Network requests
- Database queries
- Any operation that might hang

**With context (preferred):**

```go
func doWorkWithContext(ctx context.Context) <-chan string {
    result := make(chan string)
    go func() {
        select {
        case <-time.After(2 * time.Second):
            result <- "Work complete"
        case <-ctx.Done():
            result <- "Cancelled"
        }
    }()
    return result
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    result := doWorkWithContext(ctx)
    fmt.Println(<-result)
}
```

---

## Context for Cancellation

Use `context` package for cancellation, deadlines, and passing request-scoped values.

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func worker(ctx context.Context, id int) {
    for {
        select {
        case <-ctx.Done():
            fmt.Printf("Worker %d: stopping (%v)\n", id, ctx.Err())
            return
        default:
            fmt.Printf("Worker %d: working...\n", id)
            time.Sleep(500 * time.Millisecond)
        }
    }
}

func main() {
    // Create cancellable context
    ctx, cancel := context.WithCancel(context.Background())

    // Start workers
    for i := 1; i <= 3; i++ {
        go worker(ctx, i)
    }

    // Let them work for 2 seconds
    time.Sleep(2 * time.Second)

    // Cancel all workers
    fmt.Println("Cancelling workers...")
    cancel()

    // Give them time to clean up
    time.Sleep(1 * time.Second)
}
```

**Context types:**

```go
// Manual cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Deadline
deadline := time.Now().Add(5 * time.Second)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()

// With values (use sparingly!)
ctx := context.WithValue(context.Background(), "userID", 123)
```

**Use cases:**
- HTTP request handling
- Graceful shutdown
- Distributed tracing
- Request cancellation

---

## Done Channel

Signal goroutines to stop using a done channel.

```go
package main

import (
    "fmt"
    "time"
)

func worker(done <-chan bool, id int) {
    for {
        select {
        case <-done:
            fmt.Printf("Worker %d: stopping\n", id)
            return
        default:
            fmt.Printf("Worker %d: working...\n", id)
            time.Sleep(500 * time.Millisecond)
        }
    }
}

func main() {
    done := make(chan bool)

    // Start workers
    for i := 1; i <= 3; i++ {
        go worker(done, i)
    }

    // Let them work
    time.Sleep(2 * time.Second)

    // Stop all workers
    close(done)  // Closing broadcasts to all receivers

    // Give them time to stop
    time.Sleep(500 * time.Millisecond)
}
```

**Tip:** Closing a channel broadcasts to **all** receivers, making it perfect for signaling multiple goroutines.

---

## Rate Limiting

Control the rate of operations using `time.Ticker`.

### Simple Rate Limiting

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    requests := make(chan int, 5)
    for i := 1; i <= 5; i++ {
        requests <- i
    }
    close(requests)

    // Process 1 request per second
    limiter := time.Tick(1 * time.Second)

    for req := range requests {
        <-limiter  // Wait for rate limiter
        fmt.Println("Processing request", req, "at", time.Now().Format("15:04:05"))
    }
}
```

### Bursty Rate Limiting

```go
func main() {
    requests := make(chan int, 10)
    for i := 1; i <= 10; i++ {
        requests <- i
    }
    close(requests)

    // Allow bursts of 3 requests
    burstyLimiter := make(chan time.Time, 3)

    // Fill bucket with 3 slots
    for i := 0; i < 3; i++ {
        burstyLimiter <- time.Now()
    }

    // Refill 1 slot every second
    go func() {
        for t := range time.Tick(1 * time.Second) {
            burstyLimiter <- t
        }
    }()

    for req := range requests {
        <-burstyLimiter
        fmt.Println("Processing request", req, "at", time.Now().Format("15:04:05"))
    }
}
```

**Use cases:**
- API rate limiting
- Database connection throttling
- Network request throttling

---

## Semaphore

Limit concurrent access using a buffered channel as a semaphore.

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

func worker(id int, sem chan struct{}, wg *sync.WaitGroup) {
    defer wg.Done()

    // Acquire semaphore
    sem <- struct{}{}
    defer func() { <-sem }()  // Release semaphore

    fmt.Printf("Worker %d: started\n", id)
    time.Sleep(2 * time.Second)  // Simulate work
    fmt.Printf("Worker %d: finished\n", id)
}

func main() {
    const maxConcurrent = 3
    sem := make(chan struct{}, maxConcurrent)  // Semaphore with limit of 3
    var wg sync.WaitGroup

    // Start 10 workers, but only 3 can run concurrently
    for i := 1; i <= 10; i++ {
        wg.Add(1)
        go worker(i, sem, &wg)
    }

    wg.Wait()
    fmt.Println("All workers done")
}
```

**Alternative using `golang.org/x/sync/semaphore`:**

```go
import "golang.org/x/sync/semaphore"

sem := semaphore.NewWeighted(3)  // Max 3 concurrent

// Acquire
if err := sem.Acquire(ctx, 1); err != nil {
    // Handle error
}
defer sem.Release(1)
```

---

## Pub/Sub (Broadcast)

Broadcast messages to multiple subscribers.

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

type PubSub struct {
    mu          sync.RWMutex
    subscribers map[string]chan string
}

func NewPubSub() *PubSub {
    return &PubSub{
        subscribers: make(map[string]chan string),
    }
}

func (ps *PubSub) Subscribe(id string) <-chan string {
    ps.mu.Lock()
    defer ps.mu.Unlock()

    ch := make(chan string, 10)
    ps.subscribers[id] = ch
    return ch
}

func (ps *PubSub) Unsubscribe(id string) {
    ps.mu.Lock()
    defer ps.mu.Unlock()

    if ch, ok := ps.subscribers[id]; ok {
        close(ch)
        delete(ps.subscribers, id)
    }
}

func (ps *PubSub) Publish(msg string) {
    ps.mu.RLock()
    defer ps.mu.RUnlock()

    for _, ch := range ps.subscribers {
        // Non-blocking send (skip slow subscribers)
        select {
        case ch <- msg:
        default:
        }
    }
}

func subscriber(id string, ch <-chan string) {
    for msg := range ch {
        fmt.Printf("Subscriber %s received: %s\n", id, msg)
    }
}

func main() {
    ps := NewPubSub()

    // Create subscribers
    ch1 := ps.Subscribe("sub1")
    ch2 := ps.Subscribe("sub2")
    ch3 := ps.Subscribe("sub3")

    go subscriber("sub1", ch1)
    go subscriber("sub2", ch2)
    go subscriber("sub3", ch3)

    // Publish messages
    ps.Publish("Hello")
    time.Sleep(100 * time.Millisecond)
    ps.Publish("World")
    time.Sleep(100 * time.Millisecond)

    // Unsubscribe one
    ps.Unsubscribe("sub2")
    ps.Publish("After unsubscribe")

    time.Sleep(1 * time.Second)
}
```

**Use case in Connversed:**
This pattern is perfect for broadcasting messages in chat rooms!

---

## Patterns for Connversed

### Pattern 1: Broadcast with Goroutines

```go
func (r *Room) Broadcast(sender *Client, message string) {
    r.mu.RLock()
    clients := make([]*Client, len(r.Clients))
    copy(clients, r.Clients)
    r.mu.RUnlock()

    var wg sync.WaitGroup
    for _, receiver := range clients {
        wg.Add(1)
        go func(c *Client) {
            defer wg.Done()
            c.Write(message, sender)
        }(receiver)
    }
    wg.Wait()
}
```

### Pattern 2: Message Queue

```go
type Room struct {
    // ... existing fields ...
    Messages chan Message
}

func (r *Room) StartMessageProcessor() {
    go func() {
        for msg := range r.Messages {
            r.broadcast(msg)
        }
    }()
}

func (r *Room) SendMessage(msg Message) {
    r.Messages <- msg
}
```

### Pattern 3: Client Heartbeat

```go
func (c *Client) StartHeartbeat(done <-chan bool) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if err := c.Ping(); err != nil {
                log.Printf("Client %s disconnected", c.Id)
                c.Disconnect()
                return
            }
        case <-done:
            return
        }
    }
}
```

---

## Summary Table

| Pattern | Use When | Key Benefit |
|---------|----------|-------------|
| Worker Pool | Limit concurrency | Resource control |
| Pipeline | Sequential processing | Clear stages |
| Fan-Out/Fan-In | Parallel processing | Speed up computation |
| Timeout | External operations | Prevent hanging |
| Context | Cancellation needed | Propagate cancellation |
| Done Channel | Simple signaling | Easy to implement |
| Rate Limiting | Control request rate | Protect resources |
| Semaphore | Limit concurrent access | Fine-grained control |
| Pub/Sub | One-to-many messaging | Decouple publishers/subscribers |

## See Also

- [Goroutines](goroutines.md) - Concurrent execution
- [Channels](channels.md) - Communication between goroutines
- [WaitGroup](waitgroup.md) - Waiting for goroutines
- [Race Conditions](race-conditions.md) - Thread safety
