package identity

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

var (
	// ErrUsernameTaken means the username belongs to a different key.
	ErrUsernameTaken = errors.New("username belongs to a different SSH key")
	// ErrTooManyUsernames means the key has already claimed its share of names.
	ErrTooManyUsernames = errors.New("this SSH key has claimed too many usernames")
	// ErrRegistryFull means no new usernames can be claimed at all.
	ErrRegistryFull = errors.New("no more usernames can be registered")
)

// Limits on how much one key, and everyone together, can claim. They stop a
// single client from squatting every name or filling the disk.
const (
	MaxUsernamesPerKey = 3
	MaxUsernames       = 10000
)

// Registry binds each username to the first SSH public key that used it. It is
// kept in a file shaped like authorized_keys ("<username> <key>" per line) so
// that a username survives server restarts and redeploys.
type Registry struct {
	mu    sync.Mutex
	path  string
	owner map[string]ssh.PublicKey
	names []string // in claim order, so the file is stable
}

// OpenRegistry loads the registry at path, or starts an empty one if the file
// does not exist yet.
func OpenRegistry(path string) (*Registry, error) {
	r := &Registry{path: path, owner: map[string]ssh.PublicKey{}}

	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		name, keyText, ok := strings.Cut(text, " ")
		if !ok || !ValidUsername(name) {
			return nil, fmt.Errorf("%s:%d: malformed entry", path, line)
		}
		key, _, _, _, err := gossh.ParseAuthorizedKey([]byte(keyText))
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		if _, dup := r.owner[name]; !dup {
			r.names = append(r.names, name)
		}
		r.owner[name] = key
	}
	return r, scanner.Err()
}

// Check reports whether key may use name, without claiming it. A name nobody
// has claimed yet is allowed.
func (r *Registry) Check(name string, key ssh.PublicKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.check(name, key)
}

// Registered reports whether someone has claimed name.
func (r *Registry) Registered(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.owner[name]
	return ok
}

// Claim binds name to key if it is free, and succeeds without changes if key
// already owns it.
func (r *Registry) Claim(name string, key ssh.PublicKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.check(name, key); err != nil {
		return err
	}
	if _, owned := r.owner[name]; owned {
		return nil
	}
	if len(r.names) >= MaxUsernames {
		return ErrRegistryFull
	}
	claimed := 0
	for _, owner := range r.owner {
		if ssh.KeysEqual(owner, key) {
			claimed++
		}
	}
	if claimed >= MaxUsernamesPerKey {
		return ErrTooManyUsernames
	}

	r.owner[name] = key
	r.names = append(r.names, name)
	if err := r.save(); err != nil {
		delete(r.owner, name)
		r.names = r.names[:len(r.names)-1]
		return err
	}
	return nil
}

func (r *Registry) check(name string, key ssh.PublicKey) error {
	if owner, ok := r.owner[name]; ok && !ssh.KeysEqual(owner, key) {
		return ErrUsernameTaken
	}
	return nil
}

// save rewrites the file atomically so a crash never leaves it half written.
func (r *Registry) save() error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(r.path), ".connversed-users-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	w := bufio.NewWriter(tmp)
	for _, name := range r.names {
		fmt.Fprintf(w, "%s %s", name, gossh.MarshalAuthorizedKey(r.owner[name]))
	}
	if err := w.Flush(); err != nil {
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
	return os.Rename(tmp.Name(), r.path)
}
