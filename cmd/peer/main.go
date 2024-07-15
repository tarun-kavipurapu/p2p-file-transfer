package main

import (
	"log"
	"os"
	"sync"
	"tarun-kavipurapu/p2p-transfer/peer"
	"tarun-kavipurapu/p2p-transfer/pkg"
	"time"
)

func main() {

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	//this is the ip adress which the peer is listening
	opts := pkg.TransportOpts{
		ListenAddr: "127.0.0.1:8001",
		Decoder:    pkg.DefaultDecoder{},
	}
	p := peer.NewPeerServer(opts, "127.0.0.1:8000")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := p.Start()
		if err != nil {
			log.Fatalln(err)
		}

	}()

	time.Sleep(1 * time.Second)
	err := p.RegisterFile("/mnt/d/Devlopement/p2p-file-transfer/TestingFiles/test_image.tif")
	// err := p.RegisterFile("D:\\Devlopement\\go-network-Stream\\TestingFiles\\test_image.tif")
	if err != nil {
		log.Println("Error Register peer", err)
	}

	wg.Wait()
	// err = p.RequestChunkData("017934bd678d02194f615f9e27cab2a72839fc28653daaf81ac7933f2945467d")

}
