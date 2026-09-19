package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

func TestLobbyViewHeightDoesNotChangeWithContent(t *testing.T) {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "Type a message..."
	input.Width = 54
	input.Focus()

	model := Model{
		Width:        60,
		Height:       33,
		CurrentFocus: InputFocus,
		TextInput:    input,
		Viewport:     viewport.New(54, 23),
		Ready:        true,
	}

	tests := []struct {
		name     string
		input    string
		messages []string
	}{
		{name: "empty"},
		{name: "typing", input: "Hi Alice - I can see your message!"},
		{name: "messages", messages: []string{
			"alice: Hello Bob - shared lobby works!",
			"bob: Hi Alice - I can see your message!",
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := model
			current.TextInput.SetValue(test.input)
			current.Messages = test.messages

			if height := lipgloss.Height(LobbyView(current)); height != current.Height {
				t.Fatalf("view height = %d, want %d", height, current.Height)
			}
		})
	}
}
