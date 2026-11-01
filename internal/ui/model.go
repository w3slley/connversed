package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/bubbletea"
)

type focus int

const (
	None focus = iota
	InputFocus
	MessagesFocus
)

type Message struct{}

// MaxHistory is how many lines a session keeps; older ones scroll away for good.
const MaxHistory = 500

// MessageLimiter decides whether a user may send another message right now.
type MessageLimiter interface {
	Allow(username string) bool
}

type Model struct {
	Width        int
	Height       int
	CurrentFocus focus
	UserInput    string
	Messages     []string
	Session      ssh.Session
	Username     string
	Limiter      MessageLimiter
	Renderer     *lipgloss.Renderer
	Color        string
	Style        lipgloss.Style
	ErrStyle     lipgloss.Style
	TextInput    textinput.Model
	Viewport     viewport.Model
	Ready        bool
	Lobby        *Lobby
	Incoming     <-chan ChatMessage
}

// InitialModel builds the TUI for one SSH session. username must already be
// validated: it is printed in other users' terminals as is.
func InitialModel(s ssh.Session, username string, lobby *Lobby, limiter MessageLimiter) Model {
	renderer := bubbletea.MakeRenderer(s)
	incoming, color := lobby.Join()
	go func() {
		<-s.Context().Done()
		lobby.Leave(incoming)
	}()

	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Prompt = ""
	ti.Focus()
	ti.CharLimit = MaxMessageLength
	ti.Width = 50

	return Model{
		CurrentFocus: InputFocus,
		Renderer:     renderer,
		Color:        color,
		Session:      s,
		Username:     username,
		Limiter:      limiter,
		Style:        renderer.NewStyle().Foreground(lipgloss.Color("8")),
		ErrStyle:     renderer.NewStyle().Foreground(lipgloss.Color("3")),
		TextInput:    ti,
		Messages:     []string{},
		Ready:        false,
		Lobby:        lobby,
		Incoming:     incoming,
	}
}

func (m Model) Init() tea.Cmd {
	return waitForMessage(m.Incoming)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case ChatMessage:
		name := m.Renderer.NewStyle().Bold(true).Foreground(lipgloss.Color(msg.Color)).Render(msg.Username)
		m.addLine(fmt.Sprintf("%s: %s", name, msg.Text))
		return m, waitForMessage(m.Incoming)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.CurrentFocus == InputFocus {
			switch msg.String() {
			case "enter":
				text := SanitizeMessage(m.TextInput.Value())
				if text == "" {
					m.TextInput.Reset()
					return m, nil
				}
				if m.Limiter != nil && !m.Limiter.Allow(m.Username) {
					// Keep the text so it can be sent once the limit passes.
					m.addLine(m.ErrStyle.Render("You are sending messages too fast. Wait a moment and press enter again."))
					return m, nil
				}
				m.Lobby.Broadcast(ChatMessage{
					Username: m.Username,
					Color:    m.Color,
					Text:     text,
				})
				m.TextInput.Reset()
				return m, nil
			case "esc":
				m.CurrentFocus = MessagesFocus
				m.TextInput.Blur()
				return m, nil
			default:
				m.TextInput, cmd = m.TextInput.Update(msg)
				cmds = append(cmds, cmd)
			}
		} else if m.CurrentFocus == MessagesFocus {
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "esc", "i":
				m.CurrentFocus = InputFocus
				m.TextInput.Focus()
				return m, nil
			default:
				m.Viewport, cmd = m.Viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		// Calculate viewport height: total height - header (2) - input box (4) - command bar (2) - padding (2)
		viewportHeight := msg.Height - 10

		if !m.Ready {
			// Initialize viewport with proper dimensions
			m.Viewport = viewport.New(msg.Width-6, viewportHeight) // -6 for border and padding
			m.Viewport.YPosition = 0
			m.Ready = true
		} else {
			m.Viewport.Width = msg.Width - 6
			m.Viewport.Height = viewportHeight
		}

		// Reserve space for the input box frame and focus indicator.
		m.TextInput.Width = msg.Width - 6
	}

	return m, tea.Batch(cmds...)
}

// addLine appends a line to the chat, dropping the oldest past MaxHistory.
func (m *Model) addLine(line string) {
	m.Messages = append(m.Messages, line)
	if extra := len(m.Messages) - MaxHistory; extra > 0 {
		m.Messages = append([]string(nil), m.Messages[extra:]...)
	}
	m.Viewport.GotoBottom()
}

func waitForMessage(messages <-chan ChatMessage) tea.Cmd {
	return func() tea.Msg {
		return <-messages
	}
}

func (m Model) View() string {
	return LobbyView(m)
}
