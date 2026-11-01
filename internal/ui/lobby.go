package ui

import "sync"

type ChatMessage struct {
	Username string
	// Color is the ANSI color (0-255) the sender was assigned for this session.
	Color string
	Text  string
}

type subscriber struct {
	messages chan<- ChatMessage
	color    string
}

type Lobby struct {
	mu          sync.RWMutex
	subscribers map[<-chan ChatMessage]subscriber
}

func NewLobby() *Lobby {
	return &Lobby{subscribers: make(map[<-chan ChatMessage]subscriber)}
}

// Join subscribes a new user to the lobby. It returns the channel the user
// receives messages on and the color they are shown with, which is not shared
// with anyone else connected unless every color is already taken.
func (l *Lobby) Join() (<-chan ChatMessage, string) {
	messages := make(chan ChatMessage, 16)

	l.mu.Lock()
	color := randomUserColor(l.colorsInUse())
	l.subscribers[messages] = subscriber{messages: messages, color: color}
	l.mu.Unlock()

	return messages, color
}

func (l *Lobby) Leave(messages <-chan ChatMessage) {
	l.mu.Lock()
	delete(l.subscribers, messages)
	l.mu.Unlock()
}

func (l *Lobby) Broadcast(message ChatMessage) {
	l.mu.RLock()
	subscribers := make([]chan<- ChatMessage, 0, len(l.subscribers))
	for _, s := range l.subscribers {
		subscribers = append(subscribers, s.messages)
	}
	l.mu.RUnlock()

	for _, messages := range subscribers {
		messages <- message
	}
}

// colorsInUse must be called with l.mu held.
func (l *Lobby) colorsInUse() map[string]bool {
	used := make(map[string]bool, len(l.subscribers))
	for _, s := range l.subscribers {
		used[s.color] = true
	}
	return used
}
