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

	"connverse/internal/stats"
)

func testConfig(t *testing.T) Config {
	cfg := DefaultConfig()
	dir := t.TempDir()
	cfg.HostKeyPath = filepath.Join(dir, "host_ed25519")
	cfg.UsersPath = filepath.Join(dir, "users")
	cfg.StatsPath = filepath.Join(dir, "stats.json")
	return cfg
}

func start(t *testing.T, cfg Config) string {
	t.Helper()
	_, addr := startServer(t, cfg)
	return addr
}

func startServer(t *testing.T, cfg Config) (*Server, string) {
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
	t.Cleanup(func() { _ = srv.Stop(time.Second) })
	return srv, l.Addr().String()
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

// chat logs in, opens the lobby and sends one message through the TUI.
func chat(t *testing.T, client *gossh.Client, text string) {
	t.Helper()
	session, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	var out syncBuffer
	session.Stdout = &out
	stdin, err := session.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := session.RequestPty("xterm-256color", 30, 100, gossh.TerminalModes{}); err != nil {
		t.Fatal(err)
	}
	if err := session.Shell(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, &out, "Rooms are coming next")
	// Bubble Tea reads a multi-character write as one paste, so press enter
	// separately, as a real keypress would arrive.
	if _, err := stdin.Write([]byte(text)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := stdin.Write([]byte("\r")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, &out, ": "+text) // our own message comes back through the lobby
}

func waitFor(t *testing.T, out *syncBuffer, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), want) {
		if time.Now().After(deadline) {
			t.Fatalf("never saw %q; output: %q", want, out.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// waitNobodyOnline waits for the server to notice that every session ended.
func waitNobodyOnline(t *testing.T, srv *Server) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for srv.Usage().Online != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("%d sessions still online", srv.Usage().Online)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func readStats(t *testing.T, path string) stats.Snapshot {
	t.Helper()
	s, err := stats.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return s.Snapshot()
}

func TestUsageIsCountedAndSurvivesARestart(t *testing.T) {
	cfg := testConfig(t)
	owner := newSigner(t)

	srv, addr := startServer(t, cfg)
	client, _, err := dial(addr, "wes", owner)
	if err != nil {
		t.Fatal(err)
	}
	chat(t, client, "hello there")
	client.Close()
	waitNobodyOnline(t, srv)
	if _, _, err := dial(addr, "wes", newSigner(t)); err == nil { // wrong key
		t.Fatal("another key logged in as wes")
	}
	if _, _, err := dial(addr, "bad name", newSigner(t)); err == nil { // invalid name
		t.Fatal("invalid name was allowed")
	}
	if err := srv.Stop(2 * time.Second); err != nil {
		t.Fatal(err)
	}

	got := readStats(t, cfg.StatsPath)
	want := stats.Snapshot{Messages: 1, Logins: 1, Rejected: 2}
	if got != want {
		t.Fatalf("after the first run = %+v, want %+v", got, want)
	}

	// A new server keeps counting on top of the saved totals.
	srv, addr = startServer(t, cfg)
	client, _, err = dial(addr, "wes", owner)
	if err != nil {
		t.Fatal(err)
	}
	chat(t, client, "back again")
	client.Close()
	waitNobodyOnline(t, srv)
	if err := srv.Stop(2 * time.Second); err != nil {
		t.Fatal(err)
	}
	want = stats.Snapshot{Messages: 2, Logins: 2, Rejected: 2}
	if got := readStats(t, cfg.StatsPath); got != want {
		t.Fatalf("after the restart = %+v, want %+v", got, want)
	}
}

func TestRefusedConnectionsAreCounted(t *testing.T) {
	cfg := testConfig(t)
	cfg.MaxConnsPerHost = 1
	srv, addr := startServer(t, cfg)

	first, _, err := dial(addr, "wes", newSigner(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := dial(addr, "bob", newSigner(t)); err == nil {
		t.Fatal("second connection was allowed")
	}
	first.Close() // Stop waits for open connections
	if err := srv.Stop(2 * time.Second); err != nil {
		t.Fatal(err)
	}
	if got := readStats(t, cfg.StatsPath).Refused; got != 1 {
		t.Fatalf("refused = %d, want 1", got)
	}
}
