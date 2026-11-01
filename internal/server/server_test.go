package server

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func testConfig(t *testing.T) Config {
	cfg := DefaultConfig()
	dir := t.TempDir()
	cfg.HostKeyPath = filepath.Join(dir, "host_ed25519")
	cfg.UsersPath = filepath.Join(dir, "users")
	return cfg
}

func start(t *testing.T, cfg Config) string {
	t.Helper()
	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		if err := srv.Serve(l); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			t.Error(err)
		}
	}()
	t.Cleanup(func() { _ = Shutdown(srv, time.Second) })
	return l.Addr().String()
}

func newSigner(t *testing.T) gossh.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := gossh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

// dial connects as user. It records the explanation the server sends through
// keyboard-interactive auth, if any.
func dial(addr, user string, signer gossh.Signer) (*gossh.Client, string, error) {
	var mu sync.Mutex
	var explanation string
	auth := []gossh.AuthMethod{}
	if signer != nil {
		auth = append(auth, gossh.PublicKeys(signer))
	}
	auth = append(auth, gossh.KeyboardInteractive(func(_, instruction string, _ []string, _ []bool) ([]string, error) {
		mu.Lock()
		explanation = instruction
		mu.Unlock()
		return nil, nil
	}))
	client, err := gossh.Dial("tcp", addr, &gossh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), // test server only
		Timeout:         5 * time.Second,
	})
	mu.Lock()
	defer mu.Unlock()
	return client, explanation, err
}

// openLobby starts a TUI session and waits until the lobby is drawn.
func openLobby(t *testing.T, client *gossh.Client) {
	t.Helper()
	session, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	var out syncBuffer
	session.Stdout = &out
	if err := session.RequestPty("xterm-256color", 30, 100, gossh.TerminalModes{}); err != nil {
		t.Fatal(err)
	}
	if err := session.Shell(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), "Rooms are coming next") {
		if time.Now().After(deadline) {
			t.Fatalf("lobby never appeared; output: %q", out.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestUsernameBelongsToTheFirstKeyThatUsesIt(t *testing.T) {
	addr := start(t, testConfig(t))
	owner, intruder := newSigner(t), newSigner(t)

	client, _, err := dial(addr, "wes", owner)
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	openLobby(t, client)
	client.Close()

	// "Wes" is the same person as "wes".
	_, explanation, err := dial(addr, "Wes", intruder)
	if err == nil {
		t.Fatal("another key logged in as an existing user")
	}
	if !strings.Contains(explanation, `"wes" belongs to a different SSH key`) {
		t.Fatalf("explanation = %q", explanation)
	}

	client, _, err = dial(addr, "wes", owner)
	if err != nil {
		t.Fatalf("owner could not log in again: %v", err)
	}
	client.Close()
}

func TestClientWithSeveralKeysCanUseTheRightOne(t *testing.T) {
	addr := start(t, testConfig(t))
	owner, other := newSigner(t), newSigner(t)

	client, _, err := dial(addr, "wes", owner)
	if err != nil {
		t.Fatal(err)
	}
	client.Close()

	// The client offers the wrong key first; auth must move on to the next.
	client, err = gossh.Dial("tcp", addr, &gossh.ClientConfig{
		User:            "wes",
		Auth:            []gossh.AuthMethod{gossh.PublicKeys(other, owner)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatalf("login with the owner key second: %v", err)
	}
	client.Close()
}

func TestClientWithoutAKeyIsToldWhy(t *testing.T) {
	addr := start(t, testConfig(t))

	_, explanation, err := dial(addr, "wes", nil)
	if err == nil {
		t.Fatal("logged in without a key")
	}
	if !strings.Contains(explanation, "ssh-keygen") {
		t.Fatalf("explanation = %q", explanation)
	}
}

func TestUnsafeUsernamesAreRejectedWithoutBeingEchoed(t *testing.T) {
	addr := start(t, testConfig(t))

	for _, name := range []string{"eve\x1b[2J", "a b", strings.Repeat("x", 21)} {
		_, explanation, err := dial(addr, name, newSigner(t))
		if err == nil {
			t.Fatalf("%q was allowed", name)
		}
		if strings.ContainsRune(explanation, '\x1b') || strings.Contains(explanation, name) {
			t.Fatalf("explanation echoes the username: %q", explanation)
		}
		if !strings.Contains(explanation, "can't be used") {
			t.Fatalf("explanation = %q", explanation)
		}
	}
}

func TestUsernamesSurviveARestart(t *testing.T) {
	cfg := testConfig(t)
	owner := newSigner(t)

	addr := start(t, cfg)
	client, _, err := dial(addr, "wes", owner)
	if err != nil {
		t.Fatal(err)
	}
	openLobby(t, client) // the claim happens once the session starts
	client.Close()

	addr = start(t, cfg) // a new server reading the same files
	if _, _, err := dial(addr, "wes", newSigner(t)); err == nil {
		t.Fatal("after a restart, another key took the username")
	}
}

func TestConnectionsPerHostAreCapped(t *testing.T) {
	cfg := testConfig(t)
	cfg.MaxConnsPerHost = 1
	addr := start(t, cfg)
	signer := newSigner(t)

	// A failed login must give its slot back.
	if _, _, err := dial(addr, "wes", nil); err == nil {
		t.Fatal("logged in without a key")
	}

	first := eventually(t, func() (*gossh.Client, error) {
		c, _, err := dial(addr, "wes", signer)
		return c, err
	})

	if _, _, err := dial(addr, "wes", signer); err == nil {
		t.Fatal("second connection from the same host was allowed")
	}

	first.Close()
	eventually(t, func() (*gossh.Client, error) {
		c, _, err := dial(addr, "wes", signer)
		return c, err
	}).Close()
}

// eventually retries connect briefly: slots are released asynchronously when
// a connection closes.
func eventually(t *testing.T, connect func() (*gossh.Client, error)) *gossh.Client {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		c, err := connect()
		if err == nil {
			return c
		}
		if time.Now().After(deadline) {
			t.Fatalf("could not connect: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
