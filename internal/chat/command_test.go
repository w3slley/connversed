package chat

import (
	"testing"
)

func TestGetCommands(t *testing.T) {
	commands := GetCommands()

	expectedCommands := map[string]Command{
		"/join":     JOIN_ROOM,
		"/leave":    LEAVE_ROOM,
		"/list":     LIST_ROOMS,
		"/username": CHANGE_USERNAME,
		"/send":     SEND_MESSAGE,
		"/quit":     QUIT,
		"/help":     HELP,
	}

	if len(commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedCommands), len(commands))
	}

	for cmdStr, expectedCmd := range expectedCommands {
		if cmd, exists := commands[cmdStr]; !exists {
			t.Errorf("Command %s not found", cmdStr)
		} else if cmd != expectedCmd {
			t.Errorf("Expected command %s to be %v, got %v", cmdStr, expectedCmd, cmd)
		}
	}
}

func TestGetCommandFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected Command
	}{
		{
			name:     "join room command",
			message:  "/join room1",
			expected: JOIN_ROOM,
		},
		{
			name:     "leave room command",
			message:  "/leave",
			expected: LEAVE_ROOM,
		},
		{
			name:     "list rooms command",
			message:  "/list",
			expected: LIST_ROOMS,
		},
		{
			name:     "change username command",
			message:  "/username john",
			expected: CHANGE_USERNAME,
		},
		{
			name:     "help command",
			message:  "/help",
			expected: HELP,
		},
		{
			name:     "quit command",
			message:  "/quit",
			expected: QUIT,
		},
		{
			name:     "unknown command",
			message:  "/unknown",
			expected: Command(0),
		},
		{
			name:     "regular message",
			message:  "hello world",
			expected: Command(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCommandFromMessage(tt.message)
			if result != tt.expected {
				t.Errorf("getCommandFromMessage(%q) = %v, expected %v", tt.message, result, tt.expected)
			}
		})
	}
}

func TestGetCommandArgument(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "message with argument",
			message:  " room1",
			expected: "room1",
		},
		{
			name:     "message with trailing newline",
			message:  " room1\n",
			expected: "room1",
		},
		{
			name:     "message with spaces",
			message:  " my room name",
			expected: "my room name",
		},
		{
			name:     "empty argument",
			message:  " ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCommandArgument(tt.message)
			if result != tt.expected {
				t.Errorf("getCommandArgument(%q) = %q, expected %q", tt.message, result, tt.expected)
			}
		})
	}
}
