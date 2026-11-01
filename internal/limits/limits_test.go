package limits

import (
	"errors"
	"testing"
	"time"
)

func TestRateLimiterAllowsBurstThenRefills(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewRateLimiter(3, time.Second)
	l.now = func() time.Time { return now }

	for i := range 3 {
		if !l.Allow("wes") {
			t.Fatalf("message %d should be allowed", i+1)
		}
	}
	if l.Allow("wes") {
		t.Fatal("4th message in the same instant should be refused")
	}
	if !l.Allow("bob") {
		t.Fatal("limits are per user")
	}

	now = now.Add(time.Second)
	if !l.Allow("wes") {
		t.Fatal("one message should be allowed after one interval")
	}
	if l.Allow("wes") {
		t.Fatal("only one token should have refilled")
	}
}

func TestConnLimiter(t *testing.T) {
	l := NewConnLimiter(3, 2)

	r1, err := l.Acquire("1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Acquire("1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Acquire("1.1.1.1"); !errors.Is(err, ErrTooManyFromHost) {
		t.Fatalf("got %v, want ErrTooManyFromHost", err)
	}
	if _, err := l.Acquire("2.2.2.2"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Acquire("3.3.3.3"); !errors.Is(err, ErrServerFull) {
		t.Fatalf("got %v, want ErrServerFull", err)
	}

	r1()
	r1() // releasing twice must not free a second slot
	if _, err := l.Acquire("3.3.3.3"); err != nil {
		t.Fatalf("after release: %v", err)
	}
	if _, err := l.Acquire("4.4.4.4"); !errors.Is(err, ErrServerFull) {
		t.Fatalf("double release freed an extra slot: %v", err)
	}
}
