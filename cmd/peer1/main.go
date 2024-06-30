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
	p := peer.NewPeerServer("127.0.0.1:8082", pkg.Decoder{})
	// pkg server in 3000 port
	err := p.StartListener()
	if err != nil {
		log.Fatalf("Error startign peer server")
	}
	err = p.Connect(":3000", "central")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	fmt.Println("Connected to server")

	err = p.RequestFile("017934bd678d02194f615f9e27cab2a72839fc28653daaf81ac7933f2945467d")
	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Println("Message sent")

	p.CloseStream()
}
