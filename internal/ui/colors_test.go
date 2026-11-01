package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"connverse/internal/chat"
)

func TestRandomUserColorComesFromThePalette(t *testing.T) {
	for range 100 {
		if color := randomUserColor(nil); !slices.Contains(userColors, color) {
			t.Fatalf("color %q is not in the palette", color)
		}
	}
}

func TestIncomingMessageShowsSenderAndText(t *testing.T) {
	model := Model{Renderer: lipgloss.NewRenderer(nil)}

	updated, _ := model.Update(ChatMessage{Username: "alice", Color: "205", Text: "hello"})

	messages := updated.(Model).Messages
	if len(messages) != 1 || !strings.Contains(messages[0], "alice") || !strings.HasSuffix(messages[0], ": hello") {
		t.Fatalf("messages = %q", messages)
	}
}

func TestTypingQDoesNotQuit(t *testing.T) {
	input := textinput.New()
	input.Focus()
	model := Model{CurrentFocus: InputFocus, TextInput: input}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if cmd != nil && cmd() == tea.Quit() {
		t.Fatal("typing q quit the session")
	}
	if got := updated.(Model).TextInput.Value(); got != "q" {
		t.Fatalf("input = %q, want q", got)
	}
}

func TestLobbyViewOnlyAdvertisesAvailableFeatures(t *testing.T) {
	input := textinput.New()
	view := LobbyView(Model{Width: 100, Height: 30, CurrentFocus: InputFocus, TextInput: input})

	for _, want := range []string{chat.ROOMS_COMING_NEXT, chat.SEND_MESSAGE_COMMAND, chat.QUIT_UI_COMMAND} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q", want)
		}
	}
	for _, unwanted := range []string{"join room", "create room", "go to lobby"} {
		if strings.Contains(view, unwanted) {
			t.Errorf("view advertises unavailable feature %q", unwanted)
		}
	}
}

func TestLobbyViewShowsTheUsersOwnName(t *testing.T) {
	input := textinput.New()
	view := LobbyView(Model{
		Width: 100, Height: 30,
		CurrentFocus: InputFocus,
		TextInput:    input,
		Username:     "wes",
		Color:        "214",
		Renderer:     lipgloss.NewRenderer(nil),
	})
	if !strings.Contains(view, "you are") || !strings.Contains(view, "wes") {
		t.Fatalf("header does not tell the user who they are:\n%s", view)
	}
}

func TestLobbyViewNeverOverflowsTheTerminalWidth(t *testing.T) {
	// A line wider than the terminal gets wrapped by the terminal itself, which
	// shifts the whole screen down a row.
	input := textinput.New()
	input.Prompt = ""
	input.Width = 54
	input.Focus()

	for _, width := range []int{60, 70, 100} {
		for _, name := range []string{"", "wes", strings.Repeat("x", 20)} {
			model := Model{Width: width, Height: 33, CurrentFocus: InputFocus, TextInput: input,
				Username: name, Color: "214", Renderer: lipgloss.NewRenderer(nil)}
			for i, line := range strings.Split(LobbyView(model), "\n") {
				if w := lipgloss.Width(line); w > width {
					t.Errorf("width %d, username %q: line %d is %d columns wide: %q", width, name, i, w, line)
				}
			}
		}
	}
}
