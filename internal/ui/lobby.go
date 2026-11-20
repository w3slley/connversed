package ui

import (
	"sort"
	"sync"
)

type ChatMessage struct {
	Username string
	// Color is the ANSI color (0-255) the sender was assigned for this session.
	Color string
	Text  string
	// Notice marks a line from the lobby itself (someone joined or left)
	// rather than something a user typed. Username is the person it is about.
	Notice bool
}

// User is someone currently in the lobby.
type User struct {
	Name  string
	Color string
}

// subscriberBuffer is how many messages can wait for a user whose screen has
// not caught up yet. Past that, messages to that user are dropped.
const subscriberBuffer = 64

// HistorySize is how many of the latest messages a user gets when they join.
const HistorySize = 100

type subscriber struct {
	messages chan<- ChatMessage
	color    string
	username string
}

type Lobby struct {
	// OnMessage, if set, is called once for every message that is delivered
	// (after cleaning; dropped empty messages do not count). Set it before the
	// lobby is shared between goroutines.
	OnMessage func()

	mu          sync.RWMutex
	subscribers map[<-chan ChatMessage]subscriber
	history     []ChatMessage // latest messages, oldest first, at most HistorySize
}

func NewLobby() *Lobby {
	return &Lobby{subscribers: make(map[<-chan ChatMessage]subscriber)}
}

// Join subscribes a new user to the lobby. It returns the channel the user
// receives messages on and the color they are shown with, which is not shared
// with anyone else connected unless every color is already taken. username
// must already be validated; when it is not empty, the others in the lobby are
// told it joined (unless the same name is already connected in another
// session).
func (l *Lobby) Join(username string) (<-chan ChatMessage, string) {
	l.mu.Lock()
	// Room for the replayed history plus the usual buffer for new messages.
	messages := make(chan ChatMessage, subscriberBuffer+len(l.history))
	// The history is queued before the user is subscribed, under the same lock
	// Broadcast uses, so a message is never missed or delivered twice.
	for _, m := range l.history {
		messages <- m
	}
	color := randomUserColor(l.colorsInUse())
	announce := username != "" && !l.hasUser(username)
	l.subscribers[messages] = subscriber{messages: messages, color: color, username: username}
	l.mu.Unlock()

	if announce {
		l.notify(ChatMessage{Username: username, Color: color, Notice: true, Text: "joined the lobby"}, messages)
	}
	return messages, color
}

func (l *Lobby) Leave(messages <-chan ChatMessage) {
	l.mu.Lock()
	s, ok := l.subscribers[messages]
	delete(l.subscribers, messages)
	announce := ok && s.username != "" && !l.hasUser(s.username)
	l.mu.Unlock()

	if announce {
		l.notify(ChatMessage{Username: s.username, Color: s.color, Notice: true, Text: "left the lobby"}, nil)
	}
}

// Users lists everyone in the lobby, sorted by name. A name connected from
// several sessions is listed once.
func (l *Lobby) Users() []User {
	l.mu.RLock()
	seen := make(map[string]bool, len(l.subscribers))
	users := make([]User, 0, len(l.subscribers))
	for _, s := range l.subscribers {
		if s.username == "" || seen[s.username] {
			continue
		}
		seen[s.username] = true
		users = append(users, User{Name: s.username, Color: s.color})
	}
	l.mu.RUnlock()

	sort.Slice(users, func(i, j int) bool { return users[i].Name < users[j].Name })
	return users
}

// hasUser must be called with l.mu held.
func (l *Lobby) hasUser(username string) bool {
	for _, s := range l.subscribers {
		if s.username == username {
			return true
		}
	}
	return false
}

// notify delivers a lobby notice to everyone except skip, without blocking and
// without counting it as a chat message.
func (l *Lobby) notify(message ChatMessage, skip <-chan ChatMessage) {
	l.mu.RLock()
	targets := make([]chan<- ChatMessage, 0, len(l.subscribers))
	for key, s := range l.subscribers {
		if key != skip {
			targets = append(targets, s.messages)
		}
	}
	l.mu.RUnlock()

	for _, messages := range targets {
		select {
		case messages <- message:
		default:
		}
	}
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

	l.mu.Lock()
	l.history = append(l.history, message)
	if extra := len(l.history) - HistorySize; extra > 0 {
		l.history = append([]ChatMessage(nil), l.history[extra:]...)
	}
	subscribers := make([]chan<- ChatMessage, 0, len(l.subscribers))
	for _, s := range l.subscribers {
		subscribers = append(subscribers, s.messages)
	}
	l.mu.Unlock()

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
