// Package limits protects the server from floods: too many messages from one
// user, or too many connections from one address or in total.
package limits

import (
	"sync"
	"time"
)

// RateLimiter is a token bucket per key (a username): each key can send up to
// Burst messages at once, refilled at one message every Interval.
type RateLimiter struct {
	mu       sync.Mutex
	burst    float64
	interval time.Duration
	now      func() time.Time
	buckets  map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func NewRateLimiter(burst int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		burst:    float64(burst),
		interval: interval,
		now:      time.Now,
		buckets:  map[string]*bucket{},
	}
}

// Allow reports whether key may send a message now, and spends a token if so.
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens += float64(now.Sub(b.last)) / float64(l.interval)
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--

	// Full buckets carry no state worth keeping; dropping them keeps the map
	// from growing with every username ever seen.
	for k, other := range l.buckets {
		if k != key && other.tokens+float64(now.Sub(other.last))/float64(l.interval) >= l.burst {
			delete(l.buckets, k)
		}
	}
	return true
}
