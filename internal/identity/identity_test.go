package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func newKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestValidUsername(t *testing.T) {
	for _, name := range []string{"wes", "a", "bob_2", "x-y", "12345678901234567890"} {
		if !ValidUsername(name) {
			t.Errorf("%q should be valid", name)
		}
	}
	for _, name := range []string{"", "Wes", "-wes", "_wes", "w e s", "wes\x1b[2J", "wés", "123456789012345678901", "a/b"} {
		if ValidUsername(name) {
			t.Errorf("%q should be invalid", name)
		}
	}
}

func TestNormalizeUsername(t *testing.T) {
	if got := NormalizeUsername("Wes"); got != "wes" {
		t.Fatalf("got %q", got)
	}
}

func TestRegistryBindsUsernameToFirstKey(t *testing.T) {
	r, err := OpenRegistry(filepath.Join(t.TempDir(), "users"))
	if err != nil {
		t.Fatal(err)
	}
	owner, other := newKey(t), newKey(t)

	if err := r.Claim("wes", owner); err != nil {
		t.Fatal(err)
	}
	if err := r.Claim("wes", owner); err != nil {
		t.Fatalf("owner reclaiming: %v", err)
	}
	if err := r.Check("wes", other); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("check by other key = %v, want ErrUsernameTaken", err)
	}
	if err := r.Claim("wes", other); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("claim by other key = %v, want ErrUsernameTaken", err)
	}
	if err := r.Check("free", other); err != nil {
		t.Fatalf("unclaimed name: %v", err)
	}
}

func TestRegistryPersistsAcrossRestarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "users")
	owner, other := newKey(t), newKey(t)

	r, err := OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Claim("wes", owner); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}

	reopened, err := OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Check("wes", other); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("after reload, other key = %v, want ErrUsernameTaken", err)
	}
	if err := reopened.Check("wes", owner); err != nil {
		t.Fatalf("after reload, owner = %v", err)
	}
}

func TestRegistryCapsUsernamesPerKey(t *testing.T) {
	r, err := OpenRegistry(filepath.Join(t.TempDir(), "users"))
	if err != nil {
		t.Fatal(err)
	}
	key := newKey(t)
	for i := range MaxUsernamesPerKey {
		if err := r.Claim(string(rune('a'+i)), key); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Claim("onemore", key); !errors.Is(err, ErrTooManyUsernames) {
		t.Fatalf("got %v, want ErrTooManyUsernames", err)
	}
	if err := r.Claim("onemore", newKey(t)); err != nil {
		t.Fatalf("a different key should still be able to claim: %v", err)
	}
}

func TestOpenRegistryRejectsMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users")
	if err := os.WriteFile(path, []byte("wes not-a-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenRegistry(path); err == nil {
		t.Fatal("expected an error for a malformed file")
	}
}
