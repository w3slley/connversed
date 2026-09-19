package ui

import (
	"testing"
	"time"
)

func TestLobbyBroadcastsMessagesToEverySubscriber(t *testing.T) {
	lobby := NewLobby()
	alice := lobby.Join()
	bob := lobby.Join()
	message := ChatMessage{Username: "alice", Text: "hello bob"}

	lobby.Broadcast(message)

	for name, messages := range map[string]<-chan ChatMessage{
		"alice": alice,
		"bob":   bob,
	} {
		t.Run(name, func(t *testing.T) {
			select {
			case received := <-messages:
				if received != message {
					t.Fatalf("received %#v, want %#v", received, message)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for message")
			}
		})
	}
}

func TestLobbyStopsBroadcastingToUsersWhoLeave(t *testing.T) {
	lobby := NewLobby()
	alice := lobby.Join()
	bob := lobby.Join()
	lobby.Leave(bob)

	lobby.Broadcast(ChatMessage{Username: "alice", Text: "still here"})

	select {
	case <-alice:
	case <-time.After(time.Second):
		t.Fatal("remaining user did not receive message")
	}

	select {
	case message := <-bob:
		t.Fatalf("departed user received %#v", message)
	default:
	}
}
