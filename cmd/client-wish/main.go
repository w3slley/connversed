package main

import (
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"

	"connverse/internal/server"
)

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// The defaults suit local development; the container overrides them through
// the environment (see the Dockerfile).
func main() {
	cfg := server.DefaultConfig()
	host := getenv("CONNVERSED_HOST", "localhost")
	port := getenv("CONNVERSED_PORT", "23234")
	cfg.Addr = net.JoinHostPort(host, port)
	cfg.HostKeyPath = getenv("CONNVERSED_HOST_KEY_PATH", cfg.HostKeyPath)
	cfg.UsersPath = getenv("CONNVERSED_USERS_PATH", cfg.UsersPath)

	s, err := server.New(cfg)
	if err != nil {
		log.Fatal("Could not create server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	if err := server.Shutdown(s, 30*time.Second); err != nil {
		log.Error("Could not stop server", "error", err)
	}
}
