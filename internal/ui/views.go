package ui

import (
	"connverse/internal/chat"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func WelcomeView(m Model) string {
	commands := []string{
		chat.LOBBY_UI_COMMAND,
		chat.JOIN_ROOM_UI_COMMAND,
		chat.CREATE_ROOM_UI_COMMAND,
		chat.SEND_MESSAGE_UI_COMMAND,
		chat.QUIT_UI_COMMAND,
	}
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Bold(true).
		Padding(1).
		Width(m.Width).
		Align(lipgloss.Center)

	welcomeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Padding(1).
		Width(m.Width).
		Align(lipgloss.Center)

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(1, 1)

	connverseAscii := `
                                                  _
   ___ ___  _ __  _ ____   _____ _ __ ___  ___  __| |
  / __/ _ \| '_ \| '_ \ \ / / _ \ '__/ __|/ _ \/ _' |
 | (_| (_) | | | | | | \ V /  __/ |  \__ \  __/ (_| |
  \___\___/|_| |_|_| |_|\_/ \___|_|  |___/\___|\__,_|
  `
	title := titleStyle.Render(connverseAscii)
	welcome := welcomeStyle.Render(fmt.Sprintf(chat.WELCOME, m.Session.User()))

	contentHeight := lipgloss.Height(title) + lipgloss.Height(welcome) + 1 // +1 for command bar
	padHeight := (m.Height - contentHeight) / 2
	if padHeight < 0 {
		padHeight = 0
	}
	verticalPadding := strings.Repeat("\n", padHeight)

	commandBar := lipgloss.JoinHorizontal(lipgloss.Center,
		commandStyle.Render(strings.Join(commands, "   ")),
	)
	commandBar = lipgloss.NewStyle().
		Width(m.Width).
		Align(lipgloss.Center).
		Render(commandBar)

	return lipgloss.JoinVertical(lipgloss.Top,
		verticalPadding,
		title,
		welcome,
		lipgloss.PlaceVertical(m.Height-lipgloss.Height(verticalPadding)-lipgloss.Height(title)-lipgloss.Height(welcome), lipgloss.Bottom, commandBar),
	)
}

func LobbyView(m Model) string {
	// Styles
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Bold(true).
		Padding(0, 1)

	messagesBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		Width(m.Width - 4)

	inputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(m.Width - 4)

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)

	focusIndicatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	// Header
	header := headerStyle.Render("Lobby")

	// Command bar (render first to calculate height)
	commands := []string{
		chat.JOIN_ROOM_UI_COMMAND,
		chat.CREATE_ROOM_UI_COMMAND,
		chat.SEND_MESSAGE_COMMAND,
		chat.TOGGLE_FOCUS_COMMAND,
		chat.QUIT_UI_COMMAND,
	}
	commandBar := commandStyle.Render(strings.Join(commands, "   "))
	commandBar = lipgloss.NewStyle().
		Width(m.Width).
		Align(lipgloss.Center).
		Render(commandBar)

	// Input area (render to calculate height)
	focusIndicator := ""
	if m.CurrentFocus == InputFocus {
		focusIndicator = focusIndicatorStyle.Render("> ")
	} else {
		focusIndicator = "  "
	}
	inputBox := inputBoxStyle.Render(focusIndicator + m.TextInput.View())

	// Calculate available height for messages
	headerHeight := lipgloss.Height(header)
	inputHeight := lipgloss.Height(inputBox)
	commandHeight := lipgloss.Height(commandBar)
	availableHeight := m.Height - headerHeight - inputHeight - commandHeight - 2 // -2 for padding

	// Messages area
	messagesContent := ""
	if len(m.Messages) == 0 {
		messagesContent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("No messages yet. Type a message below and press Enter to send.")
	} else {
		// Join all messages
		messagesContent = strings.Join(m.Messages, "\n")
	}

	// Set viewport content
	m.Viewport.SetContent(messagesContent)
	messagesBox := messagesBoxStyle.Height(availableHeight).Render(m.Viewport.View())

	// Combine all elements
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		messagesBox,
		inputBox,
		commandBar,
	)
}
