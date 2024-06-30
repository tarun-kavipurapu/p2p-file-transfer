package main

import (
	"fmt"
	"log"
	peer "tarun-kavipurapu/stream-files/internal/peer"
	"tarun-kavipurapu/stream-files/pkg"
)

func main() {
	// pkg.RegisterTypes() // Register types once globally

	//peer server listening in 8081 port
	p := peer.NewPeerServer("127.0.0.1:8081", pkg.Decoder{})
	// pkg server in 3000 port
	err := p.StartListener()
	if err != nil {
		log.Fatalf("Error startign peer server")
	}
	err = p.ConnectServer(":3000")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	fmt.Println("Connected to server")

	err = p.RegisterPeer()

	if err != nil {
		log.Fatalf("Failed to Register Peer: %v", err)
	}
	p.RegisterFile()

	fmt.Println("Message sent")
	p.CloseStream()
	//replacing
}
