// Package server wires the SSH server: who may connect, how many, and which
// TUI they get once they are in.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	gossh "golang.org/x/crypto/ssh"

	"connverse/internal/identity"
	"connverse/internal/limits"
	"connverse/internal/stats"
	"connverse/internal/ui"
)

type Config struct {
	Addr        string
	HostKeyPath string
	// UsersPath is the file that records which SSH key owns each username.
	UsersPath string
	// StatsPath is where usage totals are saved; StatsInterval is how often
	// they are saved and logged (only when they changed).
	StatsPath     string
	StatsInterval time.Duration

	// IdleTimeout closes connections with no traffic in either direction,
	// including ones that never finish the SSH handshake.
	IdleTimeout time.Duration
	// MaxConns and MaxConnsPerHost cap concurrent connections, in total and
	// per remote IP address.
	MaxConns        int
	MaxConnsPerHost int
	// MessageBurst messages can be sent at once, then one per MessageInterval.
	MessageBurst    int
	MessageInterval time.Duration
}

// DefaultConfig holds limits suited to a small public chat.
func DefaultConfig() Config {
	return Config{
		Addr:            "localhost:23234",
		HostKeyPath:     ".ssh/id_ed25519",
		UsersPath:       ".ssh/connversed_users",
		StatsPath:       ".ssh/connversed_stats.json",
		StatsInterval:   5 * time.Minute,
		IdleTimeout:     30 * time.Minute,
		MaxConns:        200,
		MaxConnsPerHost: 5,
		MessageBurst:    5,
		MessageInterval: time.Second,
	}
}

// Server is an SSH server plus the background reporting of usage stats.
type Server struct {
	*ssh.Server
	usage    *stats.Stats
	reporter *stats.Reporter
}

func New(cfg Config) (*Server, error) {
	registry, err := identity.OpenRegistry(cfg.UsersPath)
	if err != nil {
		return nil, fmt.Errorf("loading users: %w", err)
	}
	usage, err := stats.Open(cfg.StatsPath)
	if err != nil {
		return nil, fmt.Errorf("loading stats: %w", err)
	}
	conns := limits.NewConnLimiter(cfg.MaxConns, cfg.MaxConnsPerHost)
	rate := limits.NewRateLimiter(cfg.MessageBurst, cfg.MessageInterval)
	lobby := ui.NewLobby()
	lobby.OnMessage = usage.MessageSent

	srv, err := wish.NewServer(
		wish.WithAddress(cfg.Addr),
		wish.WithHostKeyPath(cfg.HostKeyPath),
		wish.WithIdleTimeout(cfg.IdleTimeout),
		wish.WithPublicKeyAuth(publicKeyAuth(registry)),
		wish.WithKeyboardInteractiveAuth(explainRejection(registry, usage)),
		ssh.AllocatePty(),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler(lobby, rate)),
			claimUsername(registry, usage),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		return nil, err
	}
	srv.ConnCallback = limitConnections(conns, usage)
	return &Server{
		Server:   srv,
		usage:    usage,
		reporter: stats.StartReporter(usage, cfg.StatsPath, cfg.StatsInterval),
	}, nil
}

// Usage returns the current usage counters.
func (s *Server) Usage() stats.Snapshot { return s.usage.Snapshot() }

// Stop shuts the server down, giving open sessions up to timeout to finish,
// then writes the final usage numbers.
func (s *Server) Stop(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := s.Shutdown(ctx)
	s.reporter.Stop()
	if err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		return err
	}
	return nil
}

// publicKeyAuth admits a key for a username that is valid and either unclaimed
// or claimed by that same key. Rejecting here, rather than after login, lets
// an SSH client with several keys move on and try the next one.
func publicKeyAuth(registry *identity.Registry) ssh.PublicKeyHandler {
	return func(ctx ssh.Context, key ssh.PublicKey) bool {
		name := identity.NormalizeUsername(ctx.User())
		return identity.ValidUsername(name) && registry.Check(name, key) == nil
	}
}

// explainRejection runs only when public key auth failed for every key the
// client offered (or it offered none). It never lets anyone in; it shows the
// person why they were refused, which a bare "Permission denied" does not.
func explainRejection(registry *identity.Registry, usage *stats.Stats) ssh.KeyboardInteractiveHandler {
	return func(ctx ssh.Context, challenge gossh.KeyboardInteractiveChallenge) bool {
		usage.LoginRejected()
		_, _ = challenge("connversed", rejectionReason(registry, ctx.User()), nil, nil)
		return false
	}
}

func rejectionReason(registry *identity.Registry, user string) string {
	name := identity.NormalizeUsername(user)
	switch {
	case !identity.ValidUsername(name):
		// Do not echo the name back: it is attacker-controlled text.
		return fmt.Sprintf("That username can't be used. Use 1 to %d lowercase letters, digits, '-' or '_', starting with a letter or digit.", identity.MaxUsernameLength)
	case registry.Registered(name):
		return fmt.Sprintf("The username %q belongs to a different SSH key. Connect with another username.", name)
	default:
		return "connversed identifies you by your SSH key, and your SSH client did not offer one. Create one with `ssh-keygen -t ed25519` and connect again."
	}
}

// claimUsername binds the username to the session's key the first time it is
// used. Two people racing for the same new name both pass authentication, so
// this is where the loser is turned away.
func claimUsername(registry *identity.Registry, usage *stats.Stats) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			name := identity.NormalizeUsername(s.User())
			key := s.PublicKey()
			if key == nil || !identity.ValidUsername(name) {
				usage.LoginRejected()
				wish.Fatalln(s, "connversed needs a valid username and an SSH key.")
				return
			}
			switch err := registry.Claim(name, key); {
			case errors.Is(err, identity.ErrUsernameTaken):
				usage.LoginRejected()
				wish.Fatalf(s, "The username %q belongs to a different SSH key. Connect with another username.\r\n", name)
				return
			case errors.Is(err, identity.ErrTooManyUsernames):
				usage.LoginRejected()
				wish.Fatalf(s, "Your SSH key already has %d usernames. Connect with one of them.\r\n", identity.MaxUsernamesPerKey)
				return
			case err != nil:
				log.Error("could not claim username", "user", name, "error", err)
				wish.Fatalln(s, "Could not register your username, please try again later.")
				return
			}
			defer usage.LoggedIn()()
			next(s)
		}
	}
}

// limitConnections enforces the connection caps before the SSH handshake, so
// connections that never authenticate still count.
func limitConnections(conns *limits.ConnLimiter, usage *stats.Stats) ssh.ConnCallback {
	return func(ctx ssh.Context, conn net.Conn) net.Conn {
		host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			host = conn.RemoteAddr().String()
		}
		release, err := conns.Acquire(host)
		if err != nil {
			usage.ConnectionRefused()
			log.Warn("refusing connection", "host", host, "reason", err)
			return nil
		}
		go func() {
			<-ctx.Done()
			release()
		}()
		return conn
	}
}

func teaHandler(lobby *ui.Lobby, rate ui.MessageLimiter) bubbletea.Handler {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		name := identity.NormalizeUsername(s.User())
		return ui.InitialModel(s, name, lobby, rate), []tea.ProgramOption{tea.WithAltScreen()}
	}
}
