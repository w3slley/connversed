package chat

import (
	"fmt"
	"slices"
)

type Room struct {
	Id       string
	Name     string
	Clients  []*Client
	Messages []*Message
	Lobby    *Lobby
}

func (r *Room) Broadcast(sender *Client, message string) {
	for _, receiver := range r.Clients {
		receiver.Write(message, sender)
	}
}

func (r *Room) BroadcastLog(message string) {
	for _, receiver := range r.Clients {
		receiver.Log(message)
	}
}

func (r *Room) JoinClient(client *Client) {
	r.Clients = append(r.Clients, client)
	client.Room = r

	client.Log(fmt.Sprintf(JOINED_ROOM, r.Name))
}

func (r *Room) RemoveClient(client *Client) {
	indexToDelete := -1
	for i, clientInRoom := range r.Clients {
		if clientInRoom.Id == client.Id {
			indexToDelete = i
		} else {
			clientInRoom.Log(fmt.Sprintf(USER_LEFT_ROOM, client.Username))
		}
	}
	if indexToDelete != -1 {
		r.Clients = slices.Delete(r.Clients, indexToDelete, indexToDelete+1)
		client.Log(fmt.Sprintf(LEFT_ROOM, r.Name))
	}
	if len(r.Clients) == 0 {
		r.Lobby.RemoveRoom(r.Id)
	}
	client.Room = nil
}
