package main

import (
	"connverse/internal/chat"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func welcomeView(m model) string {
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
		Width(m.width).
		Align(lipgloss.Center)

	welcomeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Padding(1).
		Width(m.width).
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
	welcome := welcomeStyle.Render(fmt.Sprintf(chat.WELCOME, m.session.User()))

	contentHeight := lipgloss.Height(title) + lipgloss.Height(welcome) + 1 // +1 for command bar
	padHeight := (m.height - contentHeight) / 2
	if padHeight < 0 {
		padHeight = 0
	}
	verticalPadding := strings.Repeat("\n", padHeight)

	commandBar := lipgloss.JoinHorizontal(lipgloss.Center,
		commandStyle.Render(strings.Join(commands, "   ")),
	)
	commandBar = lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(commandBar)

	return lipgloss.JoinVertical(lipgloss.Top,
		verticalPadding,
		title,
		welcome,
		lipgloss.PlaceVertical(m.height-lipgloss.Height(verticalPadding)-lipgloss.Height(title)-lipgloss.Height(welcome), lipgloss.Bottom, commandBar),
	)
}

func lobbyView(m model) string {
	commands := []string{
		chat.JOIN_ROOM_UI_COMMAND,
		chat.CREATE_ROOM_UI_COMMAND,
		chat.SEND_MESSAGE_UI_COMMAND,
		chat.QUIT_UI_COMMAND,
	}
	return commands[0] + m.session.User()
}
