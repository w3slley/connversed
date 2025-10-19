# Go Routines

## What are Go Routines?

**Go routines** are lightweight threads managed by the Go runtime. They allow you to run functions concurrently without the overhead of traditional OS threads.

### Key Characteristics:

1. **Lightweight**: Go routines use only a few KB of stack space (vs MBs for OS threads)
2. **Easy to create**: Just use the `go` keyword before a function call
3. **Managed by Go runtime**: The Go scheduler multiplexes goroutines onto a small number of OS threads
4. **Scalable**: You can easily run thousands or even millions of goroutines

### Basic Syntax:

```go
// Normal function call (blocking - waits for completion)
doSomething()

// Goroutine (non-blocking - runs concurrently)
go doSomething()
```

## Simple Example

```go
package main

import (
    "fmt"
    "time"
)

func printNumbers() {
    for i := 1; i <= 5; i++ {
        fmt.Println(i)
        time.Sleep(100 * time.Millisecond)
    }
}

func printLetters() {
    for i := 'a'; i <= 'e'; i++ {
        fmt.Printf("%c\n", i)
        time.Sleep(100 * time.Millisecond)
    }
}

func main() {
    // Run both functions concurrently
    go printNumbers()
    go printLetters()

    // Wait for goroutines to finish (we'll use WaitGroup for this in practice)
    time.Sleep(1 * time.Second)

    fmt.Println("Done!")
}
```

**Output** (numbers and letters will be interleaved):
```
1
a
2
b
3
c
4
d
5
e
Done!
```

## Go Routines in Connversed

### 1. Server: One Goroutine Per Client

**Location**: `cmd/server/main.go:31`

```go
func main() {
    listener, err := net.Listen(TRANSPORT_PROTOCOL, CONN)
    // ... error handling ...

    lobby := chat.NewLobby()
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Println("Error: ", err)
        }

        // Spawn a new goroutine for each client connection
        go tcp.HandleClientInput(conn, lobby)
    }
}
```

**What's happening:**
- The server accepts connections in an infinite loop
- For **each client**, it spawns a new goroutine to handle that client's input
- This allows the server to handle **multiple clients simultaneously**
- Each client runs independently without blocking others

**Why it's critical:**
Without goroutines, the server could only handle ONE client at a time. Every new client would have to wait until the previous one disconnects!

### 2. Client: Separate Read/Write Goroutines

**Location**: `cmd/client/main.go:64-65`

```go
func main() {
    wg.Add(1)

    conn, err := net.Dial(PROTOCOL, CONN)
    if err != nil {
        log.Println(err)
    }

    // One goroutine for reading messages from server
    go Read(conn)

    // Another goroutine for writing messages to server
    go Write(conn)

    // Wait for goroutines (main would exit otherwise)
    wg.Wait()
}
```

**What's happening:**
- One goroutine continuously **reads** messages from the server
- Another goroutine continuously **writes** user input to the server
- They run **concurrently** - you can type while receiving messages

**Why it's important:**
Without goroutines, you'd have to wait for a server response before typing your next message (blocking I/O).

## Anonymous Goroutines

You can create goroutines with anonymous functions:

```go
// Named function
go doSomething()

// Anonymous function
go func() {
    fmt.Println("Hello from anonymous goroutine!")
}()

// Anonymous function with parameters
name := "Alice"
go func(n string) {
    fmt.Println("Hello", n)
}(name)  // Pass name as parameter to avoid race conditions
```

## Important: Goroutine Lifecycle

Goroutines don't automatically wait for completion:

```go
func main() {
    go fmt.Println("This might not print!")
    // main() exits immediately, goroutine may not run
}
```

To wait for goroutines, use:
1. **`sync.WaitGroup`** (most common - see [waitgroup.md](waitgroup.md))
2. **Channels** (see [channels.md](channels.md))
3. **`time.Sleep()`** (only for testing/demos, not production)

## Common Pitfalls

### 1. Loop Variable Capture

**Wrong:**
```go
for i := 0; i < 5; i++ {
    go func() {
        fmt.Println(i)  // All goroutines might print 5!
    }()
}
```

**Right:**
```go
for i := 0; i < 5; i++ {
    go func(n int) {
        fmt.Println(n)  // Each goroutine gets its own copy
    }(i)  // Pass i as parameter
}
```

### 2. Forgetting to Wait

**Wrong:**
```go
func main() {
    go doWork()
    // main exits, goroutine killed
}
```

**Right:**
```go
func main() {
    var wg sync.WaitGroup
    wg.Add(1)

    go func() {
        defer wg.Done()
        doWork()
    }()

    wg.Wait()  // Wait for goroutine to finish
}
```

### 3. Sharing Memory Without Synchronization

**Wrong:**
```go
counter := 0

go func() {
    counter++  // Race condition!
}()

go func() {
    counter++  // Race condition!
}()
```

**Right:** See [race-conditions.md](race-conditions.md) for solutions.

## Best Practices

1. **Always handle goroutine completion** (WaitGroup or channels)
2. **Pass data as parameters** to avoid variable capture issues
3. **Use channels for communication** between goroutines
4. **Limit goroutine count** for resource-intensive operations (use worker pools)
5. **Use context** for cancellation and timeouts
6. **Test with `-race` flag** to detect race conditions

## See Also

- [Channels](channels.md) - Communication between goroutines
- [WaitGroup](waitgroup.md) - Waiting for multiple goroutines
- [Race Conditions](race-conditions.md) - Thread safety
- [Concurrency Patterns](concurrency-patterns.md) - Common patterns
