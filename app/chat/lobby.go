package chat

import (
	"fmt"
	"log"
	"slices"

	"github.com/google/uuid"
)

type Lobby struct {
	Clients []*Client
	Rooms   []*Room
}

func (l *Lobby) Broadcast(sender *Client, message string) {
	for _, receiver := range l.Clients {
		receiver.Write(message, sender)
	}
}

func (l *Lobby) GetRoomByName(name string) *Room {
	for _, room := range l.Rooms {
		if room.Name == name {
			return room
		}
	}
	return nil
}

func (l *Lobby) RemoveClient(client *Client) {
	for i, clientInLobby := range l.Clients {
		if clientInLobby.Id == client.Id {
			l.Clients = slices.Delete(l.Clients, i, i+1)
			break
		}
	}
}

func (l *Lobby) JoinClient(client *Client) {
	l.Clients = append(l.Clients, client)
}

func (l *Lobby) NewRoom(name string) *Room {
	var clients []*Client
	room := &Room{Id: uuid.New().String(), Name: name, Clients: clients, Lobby: l}
	log.Printf(ROOM_CREATED, room.Id)
	l.Rooms = append(l.Rooms, room)
	return room
}

func (l *Lobby) RemoveRoom(id string) {
	for i, room := range l.Rooms {
		if room.Id == id {
			l.Rooms = slices.Delete(l.Rooms, i, i+1)
			break
		}
	}

}

func (l *Lobby) ListRooms(client *Client) {
	if l.Rooms == nil {
		client.Log(NO_ROOMS)
		return
	}
	for _, room := range l.Rooms {
		if client.Room.Id == room.Id {
			client.Log(CURRENT_ROOM_ICON)
		}
		client.Log(fmt.Sprintf("%s\n", room.Name))
	}
}

func (l *Lobby) Help(client *Client) {
	client.Log("\n")
	client.Log("Commands:\n")
	client.Log("/help - list of all commands\n")
	client.Log("/list - lists all the rooms\n")
	client.Log("/join <room> - joins room named <room> if it exists or creates it if not\n")
	client.Log("/username <username> - changes username to <username>\n")
	client.Log("/quit - exits the program\n")
}

func NewLobby() *Lobby {
	return &Lobby{
		Clients: []*Client{},
	}
}
