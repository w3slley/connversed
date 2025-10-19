# sync.WaitGroup

## What is WaitGroup?

`sync.WaitGroup` is like a counter that helps you wait for multiple goroutines to finish. It's one of the most common ways to synchronize goroutines.

## The Problem It Solves

Imagine you're a teacher who sends 5 students to do different tasks:
- Student 1: Get books from library
- Student 2: Fetch papers from office
- Student 3: Bring chalk from storage
- Student 4: Get projector
- Student 5: Arrange chairs

You want to wait until **all students** are done before starting class. But how do you know when they're all back?

This is exactly what `sync.WaitGroup` does for goroutines!

## Three Key Methods

1. **`Add(n)`** - "I'm expecting `n` goroutines to finish"
2. **`Done()`** - "One goroutine just finished" (decrements counter by 1)
3. **`Wait()`** - "Block here until counter reaches 0"

## Basic Example

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

func main() {
    var wg sync.WaitGroup

    // We're going to start 3 goroutines
    wg.Add(3)

    // Goroutine 1
    go func() {
        defer wg.Done()  // Call Done when this goroutine finishes
        fmt.Println("Task 1: Starting...")
        time.Sleep(1 * time.Second)
        fmt.Println("Task 1: Done!")
    }()

    // Goroutine 2
    go func() {
        defer wg.Done()
        fmt.Println("Task 2: Starting...")
        time.Sleep(2 * time.Second)
        fmt.Println("Task 2: Done!")
    }()

    // Goroutine 3
    go func() {
        defer wg.Done()
        fmt.Println("Task 3: Starting...")
        time.Sleep(3 * time.Second)
        fmt.Println("Task 3: Done!")
    }()

    fmt.Println("Waiting for all tasks to complete...")
    wg.Wait()  // BLOCKS HERE until all 3 goroutines call Done()
    fmt.Println("All tasks completed!")
}
```

**Output:**
```
Waiting for all tasks to complete...
Task 1: Starting...
Task 2: Starting...
Task 3: Starting...
Task 1: Done!
Task 2: Done!
Task 3: Done!
All tasks completed!
```

## How It Works (Step by Step)

```go
var wg sync.WaitGroup

// Counter starts at 0

wg.Add(3)
// Counter = 3

go func() { defer wg.Done(); doWork1() }()
// Still running, counter = 3

go func() { defer wg.Done(); doWork2() }()
// Still running, counter = 3

go func() { defer wg.Done(); doWork3() }()
// Still running, counter = 3

wg.Wait()  // Blocks because counter > 0

// ... time passes ...

// First goroutine finishes -> wg.Done() -> counter = 2
// Second goroutine finishes -> wg.Done() -> counter = 1
// Third goroutine finishes -> wg.Done() -> counter = 0

// Counter is 0, wg.Wait() unblocks!
fmt.Println("All done!")
```

## WaitGroup in Connversed

### Example: Client Read/Write

**Location**: `cmd/client/main.go`

```go
var wg sync.WaitGroup

func main() {
    wg.Add(1)  // Expecting 1 goroutine (could be more if we add more)

    conn, err := net.Dial(PROTOCOL, CONN)
    if err != nil {
        log.Println(err)
    }

    // Start two goroutines
    go Read(conn)   // Read from server
    go Write(conn)  // Write to server

    wg.Wait()  // Wait forever (or until one calls wg.Done())
}
```

**Note**: In this code, `wg.Add(1)` but neither `Read()` nor `Write()` call `wg.Done()`, so the program never exits. This is intentional - the client runs until terminated.

## Common Patterns

### Pattern 1: Add Before Goroutine

**Recommended approach:**

```go
var wg sync.WaitGroup

for i := 0; i < 5; i++ {
    wg.Add(1)  // Add BEFORE starting goroutine
    go func(n int) {
        defer wg.Done()
        fmt.Println(n)
    }(i)
}

wg.Wait()
```

### Pattern 2: Add in Loop

**Also valid:**

```go
var wg sync.WaitGroup
wg.Add(5)  // Add all at once

for i := 0; i < 5; i++ {
    go func(n int) {
        defer wg.Done()
        fmt.Println(n)
    }(i)
}

wg.Wait()
```

### Pattern 3: Passing WaitGroup as Pointer

When using WaitGroup in functions, **always pass by pointer**:

```go
func worker(id int, wg *sync.WaitGroup) {
    defer wg.Done()  // Notice: pointer receiver

    fmt.Printf("Worker %d starting\n", id)
    time.Sleep(time.Second)
    fmt.Printf("Worker %d done\n", id)
}

func main() {
    var wg sync.WaitGroup

    for i := 1; i <= 5; i++ {
        wg.Add(1)
        go worker(i, &wg)  // Pass pointer
    }

    wg.Wait()
}
```

**Why pointer?** Passing by value would create a copy, and calling `Done()` on the copy wouldn't affect the original.

## Real-World Example: Parallel Downloads

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

func download(url string, wg *sync.WaitGroup) {
    defer wg.Done()

    fmt.Printf("Downloading %s...\n", url)
    time.Sleep(2 * time.Second)  // Simulate download
    fmt.Printf("Finished %s\n", url)
}

func main() {
    urls := []string{
        "https://example.com/file1.zip",
        "https://example.com/file2.zip",
        "https://example.com/file3.zip",
    }

    var wg sync.WaitGroup

    for _, url := range urls {
        wg.Add(1)
        go download(url, &wg)
    }

    wg.Wait()  // Wait for all downloads to complete
    fmt.Println("All downloads complete!")
}
```

## Using WaitGroup with Channels

You can combine WaitGroup with channels for more control:

```go
func worker(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup) {
    defer wg.Done()

    for job := range jobs {
        fmt.Printf("Worker %d processing job %d\n", id, job)
        time.Sleep(time.Second)
        results <- job * 2
    }
}

func main() {
    jobs := make(chan int, 100)
    results := make(chan int, 100)
    var wg sync.WaitGroup

    // Start 3 workers
    for w := 1; w <= 3; w++ {
        wg.Add(1)
        go worker(w, jobs, results, &wg)
    }

    // Send 9 jobs
    for j := 1; j <= 9; j++ {
        jobs <- j
    }
    close(jobs)

    // Wait for workers to finish
    wg.Wait()
    close(results)

    // Collect results
    for result := range results {
        fmt.Println("Result:", result)
    }
}
```

## Common Pitfalls

### 1. Calling Add Inside Goroutine (Race Condition)

**Wrong:**
```go
var wg sync.WaitGroup

for i := 0; i < 5; i++ {
    go func() {
        wg.Add(1)  // ⚠️ Race condition!
        defer wg.Done()
        doWork()
    }()
}

wg.Wait()  // Might return before goroutines even start!
```

**Right:**
```go
var wg sync.WaitGroup

for i := 0; i < 5; i++ {
    wg.Add(1)  // Add BEFORE starting goroutine
    go func() {
        defer wg.Done()
        doWork()
    }()
}

wg.Wait()
```

### 2. Passing WaitGroup by Value

**Wrong:**
```go
func worker(wg sync.WaitGroup) {  // ⚠️ Passing by value!
    defer wg.Done()  // Calling Done on a COPY
    doWork()
}

func main() {
    var wg sync.WaitGroup
    wg.Add(1)
    go worker(wg)  // Passes a copy
    wg.Wait()  // Hangs forever!
}
```

**Right:**
```go
func worker(wg *sync.WaitGroup) {  // Pass pointer
    defer wg.Done()
    doWork()
}

func main() {
    var wg sync.WaitGroup
    wg.Add(1)
    go worker(&wg)  // Pass pointer
    wg.Wait()
}
```

### 3. Mismatched Add/Done Counts

**Wrong:**
```go
var wg sync.WaitGroup

wg.Add(5)  // Expecting 5 goroutines

for i := 0; i < 3; i++ {  // ⚠️ Only starting 3!
    go func() {
        defer wg.Done()
        doWork()
    }()
}

wg.Wait()  // Hangs forever - waiting for 2 more Done() calls!
```

**Right:**
```go
var wg sync.WaitGroup

n := 3
wg.Add(n)  // Match the actual count

for i := 0; i < n; i++ {
    go func() {
        defer wg.Done()
        doWork()
    }()
}

wg.Wait()
```

### 4. Reusing WaitGroup Without Waiting

**Wrong:**
```go
var wg sync.WaitGroup

wg.Add(1)
go doWork1(&wg)

wg.Add(1)  // ⚠️ Counter might not be 0 yet!
go doWork2(&wg)

wg.Wait()
```

**Right:**
```go
var wg sync.WaitGroup

// First batch
wg.Add(1)
go doWork1(&wg)
wg.Wait()  // Wait for first batch to complete

// Second batch
wg.Add(1)  // Safe - counter is definitely 0
go doWork2(&wg)
wg.Wait()
```

### 5. Calling Done Too Many Times

**Wrong:**
```go
var wg sync.WaitGroup
wg.Add(1)

go func() {
    defer wg.Done()
    doWork()
    wg.Done()  // ⚠️ Calling Done twice! Panic!
}()

wg.Wait()
```

**Right:**
```go
var wg sync.WaitGroup
wg.Add(1)

go func() {
    defer wg.Done()  // Only call once
    doWork()
}()

wg.Wait()
```

## Best Practices

1. **Always use `defer wg.Done()`** - Ensures Done is called even if function panics
2. **Add before starting goroutine** - Prevents race conditions
3. **Pass WaitGroup as pointer** - Never pass by value
4. **Match Add and Done counts** - Add(n) must have exactly n Done() calls
5. **Don't reuse without Wait** - Wait for counter to reach 0 before reusing
6. **Use channels for complex synchronization** - WaitGroup is simple but limited

## When NOT to Use WaitGroup

1. **When you need to receive results** - Use channels instead
2. **When you need cancellation** - Use context instead
3. **When you need complex coordination** - Consider other sync primitives or channels
4. **When you don't know the count in advance** - Use channels with close()

## Comparison with Channels

| WaitGroup | Channels |
|-----------|----------|
| Simple counter | Can pass data |
| Just wait for completion | Can communicate |
| Know count in advance | Dynamic number of goroutines |
| Lightweight | More flexible |
| `Add()`, `Done()`, `Wait()` | `<-`, `->`, `close()`, `range` |

**Use WaitGroup when:** You just need to wait for goroutines to finish

**Use Channels when:** You need communication or don't know count in advance

## See Also

- [Goroutines](goroutines.md) - Concurrent execution
- [Channels](channels.md) - Communication between goroutines
- [Race Conditions](race-conditions.md) - Thread safety
- [Concurrency Patterns](concurrency-patterns.md) - Advanced patterns
