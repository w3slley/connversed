// Package stats counts how much the chat is used. It records numbers only,
// never usernames or message text, so it is safe to log and to keep on disk.
package stats

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

// Stats holds the usage counters. All methods are safe for concurrent use and
// all of them accept a nil receiver, so code that is handed no Stats simply
// does not count.
type Stats struct {
	messages atomic.Int64
	logins   atomic.Int64
	rejected atomic.Int64
	refused  atomic.Int64
	online   atomic.Int64
}

// Snapshot is a point-in-time copy of the counters. Everything except Online
// is a running total since the very first start, kept across restarts.
type Snapshot struct {
	// Messages sent to the lobby (after cleaning; empty ones are not counted).
	Messages int64 `json:"messages"`
	// Logins that reached the lobby.
	Logins int64 `json:"logins"`
	// Rejected logins: no key, a name owned by another key, or an invalid name.
	Rejected int64 `json:"rejected"`
	// Refused connections that hit the total or per-IP connection cap.
	Refused int64 `json:"refused"`
	// Online is how many sessions are connected right now. Never saved.
	Online int64 `json:"-"`
}

func New() *Stats { return &Stats{} }

func (s *Stats) MessageSent() {
	if s != nil {
		s.messages.Add(1)
	}
}

func (s *Stats) LoginRejected() {
	if s != nil {
		s.rejected.Add(1)
	}
}

func (s *Stats) ConnectionRefused() {
	if s != nil {
		s.refused.Add(1)
	}
}

// LoggedIn counts a login and marks the session online. Call the returned
// function when the session ends.
func (s *Stats) LoggedIn() (loggedOut func()) {
	if s == nil {
		return func() {}
	}
	s.logins.Add(1)
	s.online.Add(1)
	return func() { s.online.Add(-1) }
}

func (s *Stats) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{}
	}
	return Snapshot{
		Messages: s.messages.Load(),
		Logins:   s.logins.Load(),
		Rejected: s.rejected.Load(),
		Refused:  s.refused.Load(),
		Online:   s.online.Load(),
	}
}

// Open loads the totals saved at path, or starts from zero if there is no
// file yet.
func Open(path string) (*Stats, error) {
	s := New()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var saved Snapshot
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	s.messages.Store(saved.Messages)
	s.logins.Store(saved.Logins)
	s.rejected.Store(saved.Rejected)
	s.refused.Store(saved.Refused)
	return s, nil
}

// Save writes the totals to path atomically, so a crash never leaves a
// half-written file.
func (s *Stats) Save(path string) error {
	data, err := json.Marshal(s.Snapshot())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".connversed-stats-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
