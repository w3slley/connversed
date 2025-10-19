# Race Conditions and Thread Safety

## What is a Race Condition?

A **race condition** occurs when multiple goroutines access the same memory **at the same time**, and at least one is writing to it. The final result depends on the unpredictable timing of goroutine execution.

## Example of a Race Condition

```go
package main

import (
    "fmt"
    "sync"
)

var counter = 0

func increment(wg *sync.WaitGroup) {
    defer wg.Done()
    counter = counter + 1  // ⚠️ RACE CONDITION!
}

func main() {
    var wg sync.WaitGroup

    // Start 1000 goroutines
    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go increment(&wg)
    }

    wg.Wait()
    fmt.Println("Counter:", counter)  // Expected: 1000, Actual: ???
}
```

**Expected result:** 1000
**Actual result:** 950, 982, 999, ... (different every time!) 🐛

**Why?** The operation `counter = counter + 1` is actually three steps:
1. Read `counter` value
2. Add 1 to it
3. Write new value back

When multiple goroutines do this simultaneously, they can overwrite each other's changes.

## Detecting Race Conditions

Go has a built-in **race detector**. Run your program with the `-race` flag:

```bash
# Run program with race detector
go run -race main.go

# Run tests with race detector
go test -race ./...

# Build with race detector
go build -race
```

**Example output:**
```
==================
WARNING: DATA RACE
Write at 0x00c000014098 by goroutine 7:
  main.increment()
      /path/to/main.go:8 +0x44

Previous write at 0x00c000014098 by goroutine 6:
  main.increment()
      /path/to/main.go:8 +0x44
==================
```

## Solutions to Race Conditions

### 1. Channels (The Go Way)

**Philosophy:** "Don't communicate by sharing memory; share memory by communicating."

```go
package main

import "fmt"

func main() {
    results := make(chan int, 1000)

    // Start 1000 goroutines
    for i := 0; i < 1000; i++ {
        go func() {
            results <- 1  // Send to channel
        }()
    }

    // Collect results
    counter := 0
    for i := 0; i < 1000; i++ {
        counter += <-results  // Receive from channel
    }

    fmt.Println("Counter:", counter)  // Always 1000!
}
```

**How it works:** Only one goroutine (the main goroutine) modifies `counter`. Other goroutines send data through the channel.

### 2. Mutex (Mutual Exclusion Lock)

A mutex is like a **lock** - only one goroutine can hold it at a time.

```go
package main

import (
    "fmt"
    "sync"
)

var (
    counter int
    mu      sync.Mutex  // The lock
)

func increment(wg *sync.WaitGroup) {
    defer wg.Done()

    mu.Lock()         // 🔒 Acquire lock
    counter++         // Safe: only one goroutine here at a time
    mu.Unlock()       // 🔓 Release lock
}

func main() {
    var wg sync.WaitGroup

    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go increment(&wg)
    }

    wg.Wait()
    fmt.Println("Counter:", counter)  // Always 1000!
}
```

**How it works:**
- `mu.Lock()` - "Wait until no one else has the lock, then take it"
- Code between Lock/Unlock runs **exclusively** (one goroutine at a time)
- `mu.Unlock()` - "Release the lock so others can proceed"

**Best practice:** Always use `defer` to ensure unlock:

```go
mu.Lock()
defer mu.Unlock()  // Will unlock even if panic occurs
// ... critical section ...
```

### 3. RWMutex (Read-Write Mutex)

When you have **many reads** but **few writes**, use `sync.RWMutex`:

```go
package main

import (
    "fmt"
    "sync"
)

var (
    data  map[string]int
    rwmu  sync.RWMutex
)

// Multiple goroutines can read simultaneously
func read(key string) int {
    rwmu.RLock()         // 🔒 Read lock (shared)
    defer rwmu.RUnlock() // 🔓 Read unlock
    return data[key]
}

// Only one goroutine can write at a time
func write(key string, value int) {
    rwmu.Lock()         // 🔒 Write lock (exclusive)
    defer rwmu.Unlock() // 🔓 Write unlock
    data[key] = value
}

func main() {
    data = make(map[string]int)

    var wg sync.WaitGroup

    // Many readers
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            read("counter")  // Multiple readers OK
        }()
    }

    // Few writers
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            write("counter", n)  // Only one writer at a time
        }(i)
    }

    wg.Wait()
}
```

**Key differences:**
- **RLock**: Multiple goroutines can hold simultaneously (for reading)
- **Lock**: Only one goroutine can hold (for writing), blocks readers too
- More efficient when reads are common, writes are rare

### 4. Atomic Operations

For simple operations on integers, use `sync/atomic`:

```go
package main

import (
    "fmt"
    "sync"
    "sync/atomic"
)

var counter int64  // Must be int32 or int64 for atomic

func increment(wg *sync.WaitGroup) {
    defer wg.Done()
    atomic.AddInt64(&counter, 1)  // Atomic - no race condition!
}

func main() {
    var wg sync.WaitGroup

    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go increment(&wg)
    }

    wg.Wait()
    fmt.Println("Counter:", atomic.LoadInt64(&counter))  // Always 1000!
}
```

**Atomic operations available:**
```go
// Add
atomic.AddInt64(&counter, 1)
atomic.AddInt64(&counter, -1)

// Load (read)
value := atomic.LoadInt64(&counter)

// Store (write)
atomic.StoreInt64(&counter, 100)

// Swap
old := atomic.SwapInt64(&counter, 200)

// Compare and swap
swapped := atomic.CompareAndSwapInt64(&counter, 100, 200)
```

**When to use:**
- Simple numeric operations
- Single variables
- Performance-critical code
- Faster than mutex for basic operations

**Supported types:**
- `int32`, `int64`
- `uint32`, `uint64`
- `uintptr`
- `unsafe.Pointer`

## Race Conditions in Connversed 🚨

### Problem Areas

**1. Lobby.Clients slice** (`internal/chat/lobby.go`)

```go
// Multiple goroutines modify simultaneously!
func (l *Lobby) JoinClient(client *Client) {
    l.Clients = append(l.Clients, client)  // ⚠️ RACE CONDITION
}

func (l *Lobby) RemoveClient(client *Client) {
    // ... iterate and modify l.Clients ...  // ⚠️ RACE CONDITION
}
```

**2. Lobby.Rooms slice** (`internal/chat/lobby.go`)

```go
func (l *Lobby) NewRoom(name string) *Room {
    // ...
    l.Rooms = append(l.Rooms, room)  // ⚠️ RACE CONDITION
}

func (l *Lobby) RemoveRoom(id string) {
    // ... iterate and modify l.Rooms ...  // ⚠️ RACE CONDITION
}
```

**3. Room.Clients slice** (`internal/chat/room.go`)

```go
func (r *Room) JoinClient(client *Client) {
    r.Clients = append(r.Clients, client)  // ⚠️ RACE CONDITION
}

func (r *Room) RemoveClient(client *Client) {
    // ... iterate and modify r.Clients ...  // ⚠️ RACE CONDITION
}
```

**Why these are dangerous:**
- Each client runs in its own goroutine (`cmd/server/main.go:31`)
- Multiple clients can join/leave simultaneously
- Could corrupt slices, lose clients, or crash the server

### Solution: Add Mutexes

**Fixed Lobby:**

```go
type Lobby struct {
    Clients []*Client
    Rooms   []*Room
    mu      sync.RWMutex  // Add mutex
}

func (l *Lobby) JoinClient(client *Client) {
    l.mu.Lock()  // 🔒 Lock for writing
    defer l.mu.Unlock()
    l.Clients = append(l.Clients, client)
}

func (l *Lobby) RemoveClient(client *Client) {
    l.mu.Lock()
    defer l.mu.Unlock()
    for i, clientInLobby := range l.Clients {
        if clientInLobby.Id == client.Id {
            l.Clients = slices.Delete(l.Clients, i, i+1)
            break
        }
    }
}

func (l *Lobby) GetRoomByName(name string) *Room {
    l.mu.RLock()  // 🔒 Read lock (allows multiple readers)
    defer l.mu.RUnlock()
    for _, room := range l.Rooms {
        if room.Name == name {
            return room
        }
    }
    return nil
}
```

**Fixed Room:**

```go
type Room struct {
    Id       string
    Name     string
    Clients  []*Client
    Messages []*Message
    Lobby    *Lobby
    mu       sync.RWMutex  // Add mutex
}

func (r *Room) JoinClient(client *Client) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.Clients = append(r.Clients, client)
    client.Room = r
    client.Log(fmt.Sprintf(JOINED_ROOM, r.Name))
}

func (r *Room) Broadcast(sender *Client, message string) {
    r.mu.RLock()  // Read lock - just reading Clients list
    defer r.mu.RUnlock()
    for _, receiver := range r.Clients {
        receiver.Write(message, sender)
    }
}
```

## When to Use Each Solution

| Solution | Use When | Pros | Cons |
|----------|----------|------|------|
| **Channels** | Communication needed | Idiomatic Go, composable | Can be complex |
| **Mutex** | Simple shared state | Easy to understand | Can deadlock |
| **RWMutex** | Many reads, few writes | Better performance | Slightly more complex |
| **Atomic** | Single numeric variable | Fastest | Very limited use cases |

## Common Deadlock Scenarios

### 1. Forgetting to Unlock

```go
mu.Lock()
if someCondition {
    return  // ⚠️ Forgot to unlock! Deadlock!
}
mu.Unlock()

// Fix: Use defer
mu.Lock()
defer mu.Unlock()  // Always unlocks
if someCondition {
    return  // Safe now
}
```

### 2. Locking in Wrong Order

```go
// Goroutine 1
mu1.Lock()
mu2.Lock()  // ⚠️ Deadlock if goroutine 2 locks in opposite order!
mu2.Unlock()
mu1.Unlock()

// Goroutine 2
mu2.Lock()
mu1.Lock()  // ⚠️ Deadlock!
mu1.Unlock()
mu2.Unlock()

// Fix: Always lock in same order
// Both goroutines:
mu1.Lock()
mu2.Lock()
mu2.Unlock()
mu1.Unlock()
```

### 3. Locking on Channel Operation

```go
mu.Lock()
ch <- value  // ⚠️ If channel blocks, mutex stays locked!
mu.Unlock()

// Fix: Unlock before channel operation
mu.Lock()
data := getData()
mu.Unlock()
ch <- data  // Safe - mutex already unlocked
```

## Best Practices

1. **Always use `-race` flag during testing**
   ```bash
   go test -race ./...
   ```

2. **Keep critical sections small**
   ```go
   // Bad - long critical section
   mu.Lock()
   doLotsOfWork()  // Holds lock too long
   mu.Unlock()

   // Good - minimal critical section
   mu.Lock()
   data := getCriticalData()
   mu.Unlock()
   doLotsOfWork(data)  // Lock released
   ```

3. **Use `defer` for unlocking**
   ```go
   mu.Lock()
   defer mu.Unlock()  // Guarantees unlock even on panic
   ```

4. **Prefer channels for goroutine communication**
   - Use mutexes for protecting data
   - Use channels for coordinating goroutines

5. **Document mutex protection**
   ```go
   type Server struct {
       mu      sync.Mutex
       clients []*Client  // Protected by mu
   }
   ```

6. **Avoid nested locks when possible**
   - Can lead to deadlocks
   - If needed, always lock in same order

7. **Don't copy mutexes**
   ```go
   // Wrong
   func (s Server) method() {  // Copies Server including mutex!
       s.mu.Lock()
       defer s.mu.Unlock()
   }

   // Right
   func (s *Server) method() {  // Pointer receiver
       s.mu.Lock()
       defer s.mu.Unlock()
   }
   ```

## Testing for Race Conditions

```bash
# Run specific test with race detector
go test -race -run TestConcurrentAccess

# Run all tests with race detector
go test -race ./...

# Build with race detector (slower but catches races at runtime)
go build -race

# Run the race-enabled build
./your-program
```

## Performance Comparison

| Operation | Speed | Use Case |
|-----------|-------|----------|
| No synchronization | Fastest | Single goroutine only |
| Atomic operations | Very fast | Simple counters/flags |
| RWMutex (read) | Fast | Many readers |
| Mutex | Medium | General protection |
| RWMutex (write) | Medium | Exclusive writes |
| Channels | Slower | Communication/coordination |

## See Also

- [Goroutines](goroutines.md) - Concurrent execution
- [Channels](channels.md) - Communication between goroutines
- [WaitGroup](waitgroup.md) - Waiting for goroutines
- [Concurrency Patterns](concurrency-patterns.md) - Advanced patterns
