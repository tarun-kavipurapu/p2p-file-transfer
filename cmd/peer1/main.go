package main

import (
	"log"
	"os"
	"tarun-kavipurapu/p2p-transfer/peer"
	"tarun-kavipurapu/p2p-transfer/pkg"
	"time"
)

func main() {

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	opts := pkg.TransportOpts{
		ListenAddr: "127.0.0.1:8002",
		Decoder:    pkg.DefaultDecoder{},
	}
	p := peer.NewPeerServer(opts, "127.0.0.1:8000")
	go func() {
		err := p.Start()
		if err != nil {
			log.Fatalln(err)
		}

	}()

	time.Sleep(1 * time.Second)
	// err = p.RegisterFile("/mnt/d/Devlopement/p2p-file-transfer/TestingFiles/test_image.tif")
	// err = p.RegisterFile("D:\\Devlopement\\go-network-Stream\\TestingFiles\\test_image.tif")
	// if err != nil {
	// 	log.Println("Error Register peer", err)
	// }

	err := p.RequestChunkData("017934bd678d02194f615f9e27cab2a72839fc28653daaf81ac7933f2945467d")

	if err != nil {
		log.Println("Error in Chunk Request", err)
	}
	select {}
}
