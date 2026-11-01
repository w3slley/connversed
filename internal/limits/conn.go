package limits

import (
	"errors"
	"sync"
)

var (
	ErrServerFull      = errors.New("the server is full, try again later")
	ErrTooManyFromHost = errors.New("too many connections from your address")
)

// ConnLimiter caps concurrent sessions, in total and per remote host.
type ConnLimiter struct {
	mu       sync.Mutex
	maxTotal int
	maxHost  int
	total    int
	perHost  map[string]int
}

func NewConnLimiter(maxTotal, maxPerHost int) *ConnLimiter {
	return &ConnLimiter{maxTotal: maxTotal, maxHost: maxPerHost, perHost: map[string]int{}}
}

// Acquire reserves a slot for host. Call the returned release function once
// the session ends.
func (l *ConnLimiter) Acquire(host string) (release func(), err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.total >= l.maxTotal {
		return nil, ErrServerFull
	}
	if l.perHost[host] >= l.maxHost {
		return nil, ErrTooManyFromHost
	}
	l.total++
	l.perHost[host]++

	var once sync.Once
	return func() {
		once.Do(func() {
			l.mu.Lock()
			defer l.mu.Unlock()
			l.total--
			if l.perHost[host]--; l.perHost[host] == 0 {
				delete(l.perHost, host)
			}
		})
	}, nil
}
