package http

import (
	"bufio"
	"log"
	"net"

	"connverse/app/chat"
)

func HandleClientInput(conn net.Conn, lobby *chat.Lobby) {
	client := chat.NewClient(conn)
	lobby.JoinClient(client)
	for {
		message, err := getMessage(client)
		if err != nil {
			log.Printf(chat.USER_DISCONNECTED, client.Id)
			break
		}
		chat.ProcessCommand(message, client, lobby)
	}
}

func getMessage(client *chat.Client) (string, error) {
	reader := bufio.NewReader(client.Conn)
	return reader.ReadString('\n')
}
