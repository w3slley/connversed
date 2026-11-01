package stats

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestCountersAndOnline(t *testing.T) {
	s := New()
	s.MessageSent()
	s.MessageSent()
	s.LoginRejected()
	s.ConnectionRefused()
	out1 := s.LoggedIn()
	out2 := s.LoggedIn()
	out1()

	got := s.Snapshot()
	want := Snapshot{Messages: 2, Logins: 2, Rejected: 1, Refused: 1, Online: 1}
	if got != want {
		t.Fatalf("snapshot = %+v, want %+v", got, want)
	}
	out2()
	if online := s.Snapshot().Online; online != 0 {
		t.Fatalf("online = %d, want 0", online)
	}
}

func TestNilStatsDoesNotCountOrPanic(t *testing.T) {
	var s *Stats
	s.MessageSent()
	s.LoginRejected()
	s.ConnectionRefused()
	s.LoggedIn()()
	if got := s.Snapshot(); got != (Snapshot{}) {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestTotalsSurviveARestartButOnlineDoesNot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "stats.json")
	s := New()
	s.MessageSent()
	s.LoggedIn() // still online when saved
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	want := Snapshot{Messages: 1, Logins: 1}
	if got := reopened.Snapshot(); got != want {
		t.Fatalf("after reopening = %+v, want %+v", got, want)
	}
	reopened.MessageSent()
	if got := reopened.Snapshot().Messages; got != 2 {
		t.Fatalf("counting continues from saved totals: got %d, want 2", got)
	}
}

func TestOpenWithoutAFileStartsAtZero(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot(); got != (Snapshot{}) {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestOpenRejectsACorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("expected an error")
	}
}

func TestSavedFileHoldsOnlyNumbers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	s := New()
	s.MessageSent()
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if got, want := string(data), `{"messages":1,"logins":0,"rejected":0,"refused":0}`+"\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
}

func TestConcurrentCounting(t *testing.T) {
	s := New()
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				s.MessageSent()
				s.LoggedIn()()
			}
		}()
	}
	wg.Wait()
	got := s.Snapshot()
	if got.Messages != 5000 || got.Logins != 5000 || got.Online != 0 {
		t.Fatalf("snapshot = %+v", got)
	}
}
