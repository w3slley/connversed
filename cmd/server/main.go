package main

import (
	"log"
	"net"

	"connverse/app/chat"
	"connverse/http"
)

const (
	TRANSPORT_PROTOCOL = "tcp"
	HOST               = "localhost"
	PORT               = "8000"
	CONN               = HOST + ":" + PORT
)

func main() {
	listener, err := net.Listen(TRANSPORT_PROTOCOL, CONN)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	log.Println("Listenning on " + CONN + " 🚀")
	lobby := chat.NewLobby()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error: ", err)
		}
		go http.HandleClientInput(conn, lobby)
	}
}
