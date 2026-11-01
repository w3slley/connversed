package stats

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestUntilMidnightUTC(t *testing.T) {
	tests := []struct {
		now  time.Time
		want time.Duration
	}{
		{time.Date(2026, 11, 1, 23, 0, 0, 0, time.UTC), time.Hour},
		{time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), 24 * time.Hour},
		{time.Date(2026, 12, 31, 18, 30, 0, 0, time.UTC), 5*time.Hour + 30*time.Minute},
		// The same instant expressed in another zone is the same distance away.
		{time.Date(2026, 11, 1, 20, 0, 0, 0, time.FixedZone("UTC-3", -3*3600)), time.Hour},
	}
	for _, tt := range tests {
		if got := untilMidnightUTC(tt.now); got != tt.want {
			t.Errorf("untilMidnightUTC(%v) = %v, want %v", tt.now, got, tt.want)
		}
	}
}

type logs struct {
	mu    sync.Mutex
	lines []Snapshot
}

func (l *logs) add(s Snapshot) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, s)
}

func (l *logs) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.lines)
}

func TestReporterSavesOftenButLogsOnlyOnSchedule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	s := New()
	var l logs
	// Save every 10ms; the next log is far away.
	r := startReporter(s, path, 10*time.Millisecond, func() time.Duration { return time.Hour }, l.add)

	s.MessageSent()
	deadline := time.Now().Add(2 * time.Second)
	for {
		saved, err := Open(path)
		if err == nil && saved.Snapshot().Messages == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the count was never saved")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if n := l.count(); n != 0 {
		t.Fatalf("logged %d times before the scheduled time", n)
	}

	r.Stop()
	if n := l.count(); n != 1 {
		t.Fatalf("Stop should log the final numbers once, got %d lines", n)
	}
	if got := l.lines[0].Messages; got != 1 {
		t.Fatalf("final line messages = %d, want 1", got)
	}
}

func TestReporterLogsEachTimeTheScheduleFires(t *testing.T) {
	var l logs
	r := startReporter(New(), filepath.Join(t.TempDir(), "stats.json"), time.Hour,
		func() time.Duration { return 10 * time.Millisecond }, l.add)

	deadline := time.Now().Add(2 * time.Second)
	for l.count() < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("only %d scheduled lines", l.count())
		}
		time.Sleep(5 * time.Millisecond)
	}
	r.Stop()
}

func TestStopIsIdempotent(t *testing.T) {
	r := startReporter(New(), filepath.Join(t.TempDir(), "stats.json"), time.Hour,
		func() time.Duration { return time.Hour }, func(Snapshot) {})
	r.Stop()
	r.Stop()
}
