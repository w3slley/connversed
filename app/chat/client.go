package chat

import (
	"fmt"
	"log"
	"net"

	"github.com/google/uuid"
)

type Client struct {
	Id       string
	Username string
	Color    []byte //hex
	Conn     net.Conn
	Room     *Room
}

func (c *Client) Write(message string, sender *Client) {
	if c.Conn != nil {
		c.Conn.Write([]byte(fmt.Sprintf("%s: %s", sender.Username, message)))
	}
}

func (c *Client) Log(message string) {
	if c.Conn != nil {
		c.Conn.Write([]byte(message))
	}
}

func (c *Client) IsInLobby() bool {
	return c.Room == nil
}

func (c *Client) Quit() {
	c.Conn.Close()
}

func (c *Client) JoinRoom(lobby *Lobby, roomName string) {
	Room := lobby.GetRoomByName(roomName)
	if Room == nil {
		Room = lobby.NewRoom(roomName)
	}
	if c.IsInLobby() {
		lobby.RemoveClient(c)
	} else if c.Room != nil && len(c.Room.Clients) == 1 {
		Room.Lobby.RemoveRoom(c.Room.Id)
	}
	Room.BroadcastLog(fmt.Sprintf(USER_JOINED_ROOM, c.Username))
	Room.JoinClient(c)
}

func (c *Client) LeaveRoom() {
	lobby := c.Room.Lobby
	if c.IsInLobby() {
		c.Log(NOT_IN_ROOM)
		lobby.Help(c)
	} else {
		c.Room.RemoveClient(c)
		lobby.JoinClient(c)
	}
}

func (c *Client) ChangeUsername(username string) {
	c.Username = username
	c.Log(fmt.Sprintf(NEW_USERNAME, username))
}

func NewClient(Conn net.Conn) *Client {
	client := &Client{Id: uuid.New().String(), Username: DEFAULT_USERNAME, Conn: Conn}
	log.Printf(CLIENT_CONNECTED, client.Id)
	return client
}
