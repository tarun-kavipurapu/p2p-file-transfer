package main

import (
	"log"
	"os"
	centralserver "tarun-kavipurapu/p2p-transfer/central-server"
	"tarun-kavipurapu/p2p-transfer/pkg"
)

func main() {

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	opts := pkg.TransportOpts{
		ListenAddr: "127.0.0.1:8000",
		Decoder:    pkg.DefaultDecoder{},
	}
	server := centralserver.NewCentralServer(opts)

	err := server.Start()

	if err != nil {
		log.Println("Error Starting sserver ", err)
	}

}
