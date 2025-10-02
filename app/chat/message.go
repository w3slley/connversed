package chat

import "time"

type Message struct {
	Sender  *Client
	Message string
	Time    time.Time
}
