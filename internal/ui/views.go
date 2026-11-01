package ui

import (
	"connverse/internal/chat"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	// Use the session's renderer for the username so its color matches the
	// one shown in the chat.
	userStyle := lipgloss.NewStyle()
	if m.Renderer != nil {
		userStyle = m.Renderer.NewStyle()
	}
	you := ""
	if m.Username != "" {
		you = userStyle.Bold(true).Foreground(lipgloss.Color(m.Color)).Render(m.Username)
		you = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 1).Render("· you are ") + you
	}
	roomsNote := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 1).Render("· " + chat.ROOMS_COMING_NEXT)
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerStyle.Render("Lobby"), you, roomsNote)
	if m.Width > 0 && lipgloss.Width(header) > m.Width {
		// On a narrow terminal drop the note instead of wrapping, which would
		// push the rest of the screen down a line.
		header = lipgloss.JoinHorizontal(lipgloss.Top, headerStyle.Render("Lobby"), you)
	}
	if m.Width > 0 {
		header = lipgloss.NewStyle().MaxWidth(m.Width).Render(header)
	}

	// Command bar (render first to calculate height)
	commands := []string{
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
	availableHeight := m.Height - headerHeight - inputHeight - commandHeight - messagesBoxStyle.GetVerticalFrameSize()
	if availableHeight < 0 {
		availableHeight = 0
	}

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
	m.Viewport.Height = availableHeight
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
