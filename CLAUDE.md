# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Connverse is a chat application accessible via SSH, built in Go. The application supports multiple chat rooms where users can join, send messages, and interact with each other through command-based interactions.

## Architecture

The codebase follows a modular structure with three main executables and a clear separation between application logic and transport layers:

### Executables (`cmd/`)

1. **`cmd/server/main.go`** - TCP server that listens on `localhost:8000` and handles raw TCP connections. Creates a `Lobby` instance and spawns a goroutine for each connecting client via `tcp.HandleClientInput()`.

2. **`cmd/client/main.go`** - Basic TCP client that connects to the server. Uses separate goroutines for reading from and writing to the connection.

3. **`cmd/client-wish/main.go`** - SSH-based client built with Charm's Wish/Bubbletea libraries. Thin entry point: reads `CONNVERSED_HOST`, `CONNVERSED_PORT`, `CONNVERSED_HOST_KEY_PATH` and `CONNVERSED_USERS_PATH` (default `localhost:23234`, `.ssh/id_ed25519`, `.ssh/connversed_users`) and starts the SSH server from `internal/server`. It currently offers a single shared lobby; it does not yet connect to the TCP server. Rooms are the next feature.

### Internal Packages (`internal/`)

#### Core Chat Module (`internal/chat/`)

The chat logic is centralized in this package with the following key components:

- **`Client`** - Represents a connected user with ID, username, color, connection, and current room reference. Handles writing messages, logging, room joining/leaving, and username changes.

- **`Room`** - Represents a chat room with ID, name, clients list, messages, and reference to parent lobby. Handles broadcasting messages to all clients in the room and client join/remove operations. Automatically deletes itself when the last client leaves.

- **`Lobby`** - The global state manager that maintains all clients and rooms. Handles room creation, client management, and broadcasting. When clients aren't in a room, they're in the lobby.

- **`Command`** - Command processing system that parses user input strings (format: `/<command> <argument>`). Available commands:
  - `/join <room>` - Join or create a room
  - `/leave` - Leave current room
  - `/list` - List all rooms
  - `/username <name>` - Change username
  - `/quit` - Disconnect
  - `/help` - Show available commands
  - Regular messages (non-commands) are broadcast to current room or lobby

- **`Message`** - Message data structure (stored but not yet fully utilized)

#### SSH Server (`internal/server/`)

- **`server.go`** - Builds the Wish SSH server. Public key auth only: a username belongs to the first key that uses it (`internal/identity`). When auth fails, a keyboard-interactive handler shows the client why (no key, name owned by another key, invalid name) and never admits anyone. Connection caps are enforced in `ConnCallback`, before the handshake; there is an idle timeout. `server_test.go` holds end-to-end tests that dial a real server.

#### Identity (`internal/identity/`)

- **`username.go`** - Usernames are lowercased and must match `[a-z0-9][a-z0-9_-]*`, max 20 chars, so they can never carry terminal escape sequences.
- **`registry.go`** - Username to public key bindings, persisted in an authorized_keys-like file (`<name> <key>` per line, mode 600, written atomically). Caps: 3 names per key, 10000 in total.

#### Limits (`internal/limits/`)

- **`rate.go`** - Per-username token bucket for messages (burst 5, 1/s by default).
- **`conn.go`** - Concurrent connection caps, total (200) and per remote IP (5).

#### Transport Layer (`internal/transport/tcp/`)

- **`handler.go`** - Connection handler that creates a `Client` from the TCP connection, adds it to the lobby, and loops reading messages and processing them as commands.

#### UI Layer (`internal/ui/`)

- **`model.go`** - Bubbletea model implementing the TUI for the wish client. Manages focus (input vs messages) and user input. There is no welcome screen: sessions start in the lobby.
- **`lobby.go`** - In-memory shared lobby that broadcasts messages and hands each connected user a color that is not in use by anyone else (random from `colors.go`). `Broadcast` sanitizes text and never blocks: a user whose buffer is full misses messages instead of stalling everyone.
- **`sanitize.go`** - Strips control characters (so no ANSI escapes), bidi overrides and newlines from messages, and caps them at 256 characters. Everything printed into another user's terminal must go through it or be a validated username.
- **`views.go`** - `LobbyView`, which only advertises features that exist today.

### Data Flow

1. Server accepts connection → creates Client → adds to Lobby
2. Client sends message → handler reads it → `ProcessCommand()` parses and executes
3. Commands either:
   - Modify client/room state (join, leave, username change)
   - Broadcast messages to room/lobby participants
   - Send information back to requesting client

## Development Commands

### Running the Application

```bash
# Start the server
go run cmd/server/main.go

# Run the basic TCP client
go run cmd/client/main.go

# Run the SSH/TUI client (Wish)
go run cmd/client-wish/main.go
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./internal/chat

# Run tests with verbose output
go test -v ./internal/chat

# Run a specific test
go test -run TestGetCommands ./internal/chat
```

### Building

```bash
# Build the server
go build -o bin/server cmd/server/main.go

# Build the basic client
go build -o bin/client cmd/client/main.go

# Build the Wish client
go build -o bin/client-wish cmd/client-wish/main.go
```

## Key Dependencies

- **Charm Libraries**: bubbletea (TUI framework), lipgloss (styling), wish (SSH server), ssh (SSH protocol)
- **google/uuid**: UUID generation for clients and rooms
- **charmbracelet/log**: Structured logging

## Important Notes

- The project uses Go modules with `go 1.22.1`
- The module name is `connverse` (used in import paths)
- Client IDs and Room IDs are UUIDs
- Default username is "anonymous" until changed
- Rooms are automatically created on first `/join` and deleted when empty
- The Wish client (client-wish) is incomplete - only the lobby is implemented (rooms come next)
- `Dockerfile` builds `cmd/client-wish` into a small image for Coolify. Expose TCP port 23234 and persist `/data/.ssh` so the SSH host key and the username registry survive redeploys
