package main

import (
	"log"
	"tarun-kavipurapu/stream-files/internal/server"
	"tarun-kavipurapu/stream-files/pkg"
)

func main() {

	// wg.Add(1) // Adding two goroutines to the wait group
	// pkg.RegisterTypes() // Register types once globally
	decoder := pkg.Decoder{}

	// t.ConsumeMessages()
	s := server.NewCentralServer("127.0.0.1:3000", decoder)
	err := s.Start()
	if err != nil {
		log.Fatal(err)
	}

	select {}

}
