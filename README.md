# connversed

A chat application accessible via SSH!

```bash
ssh -p 2222 yourname@connversed.apps.longitood.com
```

Your SSH key is your identity: the first key to log in with a username owns it. Run `ssh-keygen -t ed25519` first if you don't have a key yet.

## Roadmap

- **Rooms.** The TUI has a single shared lobby for now; rooms are next.
- **Testing and CI.** See [docs/improvements.md](docs/improvements.md).

## Development

```bash
go run cmd/client-wish/main.go   # SSH server on localhost:23234
go test ./...
```

See `CLAUDE.md` for the architecture and configuration (environment variables, Docker).
