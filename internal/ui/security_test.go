package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestSanitizeMessage(t *testing.T) {
	tests := map[string]struct{ in, want string }{
		"plain":         {"hello bob", "hello bob"},
		"ansi escape":   {"hi \x1b[2J\x1b[31mred", "hi [2J[31mred"},
		"osc title":     {"\x1b]0;pwned\x07hi", "]0;pwnedhi"},
		"c1 csi":        {"a\u009b2Jb", "a2Jb"},
		"newlines":      {"line1\nline2\r\nline3", "line1 line2  line3"},
		"bidi override": {"abc‮txt.exe", "abctxt.exe"},
		"trim":          {"   spaced   ", "spaced"},
		"only controls": {"\x1b\x07\x00", ""},
		"emoji kept":    {"hi 👋🏽 👨‍👩‍👧", "hi 👋🏽 👨‍👩‍👧"},
		"accents kept":  {"olá, você está?", "olá, você está?"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := SanitizeMessage(tt.in); got != tt.want {
				t.Fatalf("SanitizeMessage(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeMessageCapsLength(t *testing.T) {
	if got := SanitizeMessage(strings.Repeat("é", MaxMessageLength+50)); len([]rune(got)) != MaxMessageLength {
		t.Fatalf("length = %d runes, want %d", len([]rune(got)), MaxMessageLength)
	}
}

func TestBroadcastSanitizesText(t *testing.T) {
	lobby := NewLobby()
	ch, _ := lobby.Join()
	lobby.Broadcast(ChatMessage{Username: "eve", Text: "\x1b[2Jgotcha"})
	if got := (<-ch).Text; strings.ContainsRune(got, '\x1b') {
		t.Fatalf("escape reached a subscriber: %q", got)
	}
}

func TestBroadcastDropsEmptyMessages(t *testing.T) {
	lobby := NewLobby()
	ch, _ := lobby.Join()
	lobby.Broadcast(ChatMessage{Username: "eve", Text: "\x1b\x07  "})
	select {
	case m := <-ch:
		t.Fatalf("empty message was delivered: %#v", m)
	default:
	}
}

func TestBroadcastDoesNotBlockOnUserWhoIsNotReading(t *testing.T) {
	lobby := NewLobby()
	stalled, _ := lobby.Join() // never read from
	active, _ := lobby.Join()

	done := make(chan struct{})
	go func() {
		for range subscriberBuffer * 3 {
			lobby.Broadcast(ChatMessage{Username: "bob", Text: "hi"})
			<-active
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("broadcast blocked on a user who is not reading")
	}
	if len(stalled) != subscriberBuffer {
		t.Fatalf("stalled user has %d queued, want a full buffer of %d", len(stalled), subscriberBuffer)
	}
}

type denyAll struct{}

func (denyAll) Allow(string) bool { return false }

func TestRateLimitedMessageIsKeptAndNotSent(t *testing.T) {
	lobby := NewLobby()
	other, _ := lobby.Join()
	input := textinput.New()
	input.Focus()
	input.SetValue("hello")
	m := Model{
		CurrentFocus: InputFocus,
		TextInput:    input,
		Lobby:        lobby,
		Username:     "wes",
		Limiter:      denyAll{},
		Renderer:     lipgloss.NewRenderer(nil),
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)

	select {
	case msg := <-other:
		t.Fatalf("rate-limited message was broadcast: %#v", msg)
	default:
	}
	if got.TextInput.Value() != "hello" {
		t.Fatalf("input = %q, want the unsent text kept", got.TextInput.Value())
	}
	if len(got.Messages) != 1 || !strings.Contains(got.Messages[0], "too fast") {
		t.Fatalf("messages = %q, want a rate-limit notice", got.Messages)
	}
}

func TestHistoryIsCapped(t *testing.T) {
	m := Model{Renderer: lipgloss.NewRenderer(nil)}
	for range MaxHistory + 10 {
		updated, _ := m.Update(ChatMessage{Username: "bob", Color: "39", Text: "hi"})
		m = updated.(Model)
	}
	if len(m.Messages) != MaxHistory {
		t.Fatalf("history = %d lines, want %d", len(m.Messages), MaxHistory)
	}
}
