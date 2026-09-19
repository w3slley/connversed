package ui

import "sync"

type ChatMessage struct {
	Username string
	Text     string
}

type Lobby struct {
	mu          sync.RWMutex
	subscribers map[<-chan ChatMessage]chan<- ChatMessage
}

func NewLobby() *Lobby {
	return &Lobby{subscribers: make(map[<-chan ChatMessage]chan<- ChatMessage)}
}

func (l *Lobby) Join() <-chan ChatMessage {
	messages := make(chan ChatMessage, 16)

	l.mu.Lock()
	l.subscribers[messages] = messages
	l.mu.Unlock()

	return messages
}

func (l *Lobby) Leave(messages <-chan ChatMessage) {
	l.mu.Lock()
	delete(l.subscribers, messages)
	l.mu.Unlock()
}

func (l *Lobby) Broadcast(message ChatMessage) {
	l.mu.RLock()
	subscribers := make([]chan<- ChatMessage, 0, len(l.subscribers))
	for _, messages := range l.subscribers {
		subscribers = append(subscribers, messages)
	}
	l.mu.RUnlock()

	for _, messages := range subscribers {
		messages <- message
	}
}
