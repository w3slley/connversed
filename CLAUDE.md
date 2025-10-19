# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Connverse is a chat application accessible via SSH, built in Go. The application supports multiple chat rooms where users can join, send messages, and interact with each other through command-based interactions.

## Architecture

The codebase follows a modular structure with three main executables and a clear separation between application logic and transport layers:

### Executables (`cmd/`)

1. **`cmd/server/main.go`** - TCP server that listens on `localhost:8000` and handles raw TCP connections. Creates a `Lobby` instance and spawns a goroutine for each connecting client via `tcp.HandleClientInput()`.

2. **`cmd/client/main.go`** - Basic TCP client that connects to the server. Uses separate goroutines for reading from and writing to the connection.

3. **`cmd/client-wish/main.go`** - SSH-based client built with Charm's Wish/Bubbletea libraries. Runs an SSH server on `localhost:23234` with a TUI interface. Note: Currently only displays the welcome screen - does not yet connect to the TCP server.

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

#### Transport Layer (`internal/transport/tcp/`)

- **`handler.go`** - Connection handler that creates a `Client` from the TCP connection, adds it to the lobby, and loops reading messages and processing them as commands.

#### UI Layer (`internal/ui/`)

- **`model.go`** - Bubbletea model implementing the TUI for the wish client. Manages screens (Welcome, Lobby, Room), focus states, and user input.
- **`views.go`** - Rendering functions for different screens (`WelcomeView`, `LobbyView`).

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
- The Wish client (client-wish) is incomplete - only welcome and lobby views are implemented
