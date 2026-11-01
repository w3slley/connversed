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

type Model struct {
	Width        int
	Height       int
	CurrentFocus focus
	UserInput    string
	Messages     []string
	Session      ssh.Session
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

func InitialModel(s ssh.Session, lobby *Lobby) Model {
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
	ti.CharLimit = 256
	ti.Width = 50

	return Model{
		CurrentFocus: InputFocus,
		Renderer:     renderer,
		Color:        color,
		Session:      s,
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
		m.Messages = append(m.Messages, fmt.Sprintf("%s: %s", name, msg.Text))
		m.Viewport.GotoBottom()
		return m, waitForMessage(m.Incoming)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.CurrentFocus == InputFocus {
			switch msg.String() {
			case "enter":
				if m.TextInput.Value() != "" {
					m.Lobby.Broadcast(ChatMessage{
						Username: m.Session.User(),
						Color:    m.Color,
						Text:     m.TextInput.Value(),
					})
					m.TextInput.Reset()
				}
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

func waitForMessage(messages <-chan ChatMessage) tea.Cmd {
	return func() tea.Msg {
		return <-messages
	}
}

func (m Model) View() string {
	return LobbyView(m)
}
