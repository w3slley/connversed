package ui

import "sync"

type ChatMessage struct {
	Username string
	// Color is the ANSI color (0-255) the sender was assigned for this session.
	Color string
	Text  string
}

// subscriberBuffer is how many messages can wait for a user whose screen has
// not caught up yet. Past that, messages to that user are dropped.
const subscriberBuffer = 64

type subscriber struct {
	messages chan<- ChatMessage
	color    string
}

type Lobby struct {
	// OnMessage, if set, is called once for every message that is delivered
	// (after cleaning; dropped empty messages do not count). Set it before the
	// lobby is shared between goroutines.
	OnMessage func()

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
	messages := make(chan ChatMessage, subscriberBuffer)

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

// Broadcast sends message to everyone in the lobby. The text is sanitized
// here, so nothing reaches another user's terminal unfiltered, and empty
// messages are dropped. Broadcast never blocks: a user who is not reading (for
// example a client that stopped reading from its connection) misses messages
// instead of freezing the chat for everybody else.
func (l *Lobby) Broadcast(message ChatMessage) {
	message.Text = SanitizeMessage(message.Text)
	if message.Text == "" {
		return
	}
	if l.OnMessage != nil {
		l.OnMessage()
	}

	l.mu.RLock()
	subscribers := make([]chan<- ChatMessage, 0, len(l.subscribers))
	for _, s := range l.subscribers {
		subscribers = append(subscribers, s.messages)
	}
	l.mu.RUnlock()

	for _, messages := range subscribers {
		select {
		case messages <- message:
		default:
		}
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
