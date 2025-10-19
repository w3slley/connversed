package chat

import (
	"strings"
)

const (
	DEFAULT_USERNAME = "anonymous"
)

type Command int

const (
	JOIN_ROOM Command = iota + 1
	LEAVE_ROOM
	LIST_ROOMS
	CHANGE_USERNAME
	SEND_MESSAGE
	QUIT
	HELP
)

const (
	LOBBY_UI_COMMAND        = "l - go to lobby"
	JOIN_ROOM_UI_COMMAND    = "j - join room"
	CREATE_ROOM_UI_COMMAND  = "c - create room"
	SEND_MESSAGE_UI_COMMAND = "s - send message"
	QUIT_UI_COMMAND         = "q - quit"
)

func GetCommands() map[string]Command {
	return map[string]Command{
		"/join":     JOIN_ROOM,
		"/leave":    LEAVE_ROOM,
		"/list":     LIST_ROOMS,
		"/username": CHANGE_USERNAME,
		"/send":     SEND_MESSAGE,
		"/quit":     QUIT,
		"/help":     HELP,
	}
}

func ProcessCommand(message string, client *Client, lobby *Lobby) {
	command := getCommandFromMessage(message)
	argument := getCommandArgument(message)
	switch command {
	case JOIN_ROOM:
		client.JoinRoom(lobby, argument)

	case LEAVE_ROOM:
		client.LeaveRoom()

	case LIST_ROOMS:
		lobby.ListRooms(client)

	case SEND_MESSAGE:
		//TODO: Implement

	case CHANGE_USERNAME:
		client.ChangeUsername(argument)

	case HELP:
		lobby.Help(client)

	case QUIT:
		client.Quit()

	default:
		if client.IsInLobby() {
			lobby.Broadcast(client, message)
		} else {
			client.Room.Broadcast(client, message)
		}
	}
}

/* Example of a valid command: /<command> <argument> */
func getCommandFromMessage(message string) Command {
	cmdStr := strings.Split(message, " ")[0]
	command := GetCommands()[cmdStr]
	return command
}

// Example: /username test -> test
func getCommandArgument(message string) string {
	return strings.TrimSuffix(strings.TrimPrefix(message, " "), "\n")
}
