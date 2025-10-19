package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/bubbletea"
)

type screen int
type focus int

const (
	WelcomeScreen screen = iota
	LobbyScreen
	RoomScreen
)

const (
	None focus = iota
	InputFocus
	MessagesFocus
)

type Message struct{}

type Model struct {
	Width         int
	Height        int
	CurrentScreen screen
	CurrentFocus  focus
	UserInput     string
	Messages      *[]Message
	Session       ssh.Session
	Style         lipgloss.Style
	ErrStyle      lipgloss.Style
}

func InitialModel(s ssh.Session) Model {
	renderer := bubbletea.MakeRenderer(s)
	return Model{
		CurrentScreen: WelcomeScreen,
		CurrentFocus:  None,
		Session:       s,
		Style:         renderer.NewStyle().Foreground(lipgloss.Color("8")),
		ErrStyle:      renderer.NewStyle().Foreground(lipgloss.Color("3")),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "c":
		case "j":
		case "s":
		case "u":
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}

	return m, nil
}

func (m Model) View() string {
	switch m.CurrentScreen {
	case WelcomeScreen:
		return WelcomeView(m)
	case LobbyScreen:
		return LobbyView(m)
	}
	return ""
}
