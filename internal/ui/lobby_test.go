package ui

import (
	"testing"
	"time"
)

func TestLobbyBroadcastsMessagesToEverySubscriber(t *testing.T) {
	lobby := NewLobby()
	alice, _ := lobby.Join()
	bob, _ := lobby.Join()
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
	alice, _ := lobby.Join()
	bob, _ := lobby.Join()
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

func TestLobbyGivesConnectedUsersDistinctColors(t *testing.T) {
	lobby := NewLobby()
	seen := map[string]bool{}

	for range userColors {
		_, color := lobby.Join()
		if seen[color] {
			t.Fatalf("color %q was handed out twice", color)
		}
		seen[color] = true
	}

	// Every color is taken now, so a further user still gets one.
	if _, color := lobby.Join(); color == "" {
		t.Fatal("no color assigned once the palette is exhausted")
	}
}

func TestLobbyReusesColorsOfUsersWhoLeave(t *testing.T) {
	lobby := NewLobby()
	var first <-chan ChatMessage
	var firstColor string
	for i := range userColors {
		ch, color := lobby.Join()
		if i == 0 {
			first, firstColor = ch, color
		}
	}
	lobby.Leave(first)

	if _, color := lobby.Join(); color != firstColor {
		t.Fatalf("color = %q, want freed color %q", color, firstColor)
	}
}
