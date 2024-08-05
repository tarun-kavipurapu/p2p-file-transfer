package centralserver

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"sync"
	"tarun-kavipurapu/p2p-transfer/pkg"
)

func init() {
	gob.Register(pkg.FileMetaData{})
	gob.Register(pkg.ChunkMetadata{})
	gob.Register(pkg.RequestChunkData{})
	gob.Register(pkg.ChunkRequestToPeer{})
	gob.Register(pkg.RegisterSeeder{})

}

type CentralServer struct {
	mu        sync.Mutex
	peers     map[string]pkg.Peer //conn
	Transport pkg.Transport
	files     map[string]*pkg.FileMetaData //fileId-->fileMetadata
	quitch    chan struct{}
}

func NewCentralServer(opts pkg.TransportOpts) *CentralServer {
	transport := pkg.NewTCPTransport(opts)

	centralServer := CentralServer{
		peers:     make(map[string]pkg.Peer),
		Transport: transport,
		files:     make(map[string]*pkg.FileMetaData),
		quitch:    make(chan struct{}),
	}
	transport.OnPeer = centralServer.OnPeer

	return &centralServer

}

func (c *CentralServer) Start() error {
	fmt.Printf("[%s] starting CentralServer...\n", c.Transport.Addr())

	err := c.Transport.ListenAndAccept("central-server-peer")
	if err != nil {
		return err
	}
	c.loop()
	return nil
}

func (c *CentralServer) loop() {

	defer func() {
		log.Println("Central Serevr has stopped due to error or quit acction")

		c.Transport.Close()
	}()
	for {
		select {
		case msg := <-c.Transport.Consume():
			// var dataMsg pkg.DataMessage
			dataMsg := new(pkg.DataMessage)
			// log.Println(msg.Payload)

			decoder := gob.NewDecoder(bytes.NewReader(msg.Payload))
			err := decoder.Decode(&dataMsg)
			if err != nil {
				log.Println("Decode Error", err.Error())
			}
			if err := c.handleMessage(msg.From, dataMsg); err != nil {
				log.Println("handle message error:", err)
			}
		case <-c.quitch:
			return
		}
	}
}

func (c *CentralServer) handleMessage(from string, dataMsg *pkg.DataMessage) error {
	switch v := dataMsg.Payload.(type) {
	//there is no need to handle Peer Registration as it is handled by the onPeer

	case pkg.FileMetaData:
		return c.handleRegisterFile(from, v)
	case pkg.RequestChunkData:

		return c.handleRequestChunkData(from, v)

	case pkg.RegisterSeeder:
		return c.registerSeeder(v)

	}
	return nil
}

func (c *CentralServer) registerSeeder(msg pkg.RegisterSeeder) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fileMetadata, exists := c.files[msg.FileId]
	if !exists {
		return fmt.Errorf("file with ID %s not found", msg.FileId)
	}

	for _, chunkMetadata := range fileMetadata.ChunkInfo {
		// Check if the peer is already in the list
		peerExists := false
		for _, peer := range chunkMetadata.PeersWithChunk {
			if peer == msg.PeerAddr {
				peerExists = true
				break
			}
		}

		// If the peer is not in the list, add it
		if !peerExists {
			chunkMetadata.PeersWithChunk = append(chunkMetadata.PeersWithChunk, msg.PeerAddr)
		}
	}

	log.Printf("Registered peer %s as a new seeder for file %s\n", msg.PeerAddr, msg.FileId)
	return nil
}
func (c *CentralServer) handleRequestChunkData(from string, msg pkg.RequestChunkData) error {
	fileId := msg.FileId
	c.mu.Lock()
	fileMetadata := c.files[fileId]
	peer := c.peers[from]
	c.mu.Unlock()

	if fileMetadata == nil {
		log.Printf("No file metadata found for file ID: %s\n", fileId)
		return fmt.Errorf("file metadata not found")
	}

	log.Println(msg)

	dataMessage := pkg.DataMessage{
		Payload: pkg.FileMetaData{
			FileId:        fileMetadata.FileId,
			FileName:      fileMetadata.FileName,
			FileExtension: fileMetadata.FileExtension,
			FileSize:      fileMetadata.FileSize,
			ChunkSize:     fileMetadata.ChunkSize,
			ChunkInfo:     fileMetadata.ChunkInfo,
		},
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(dataMessage); err != nil {
		log.Printf("Failed to encode data message: %v\n", err)
		return err
	}

	dataToSend := buf.Bytes()
	log.Printf("Sending %d bytes to peer %s\n", len(dataToSend), peer.RemoteAddr())

	//sending the responsse back to the client
	err := peer.Send(dataToSend)
	if err != nil {
		log.Printf("Failed to send data to peer %s: %v\n", peer.RemoteAddr(), err)
		return err
	}

	log.Println("Data sent successfully to peer", peer.RemoteAddr())
	return nil
}

func (c *CentralServer) handleRegisterFile(from string, msg pkg.FileMetaData) error {

	c.mu.Lock()
	defer c.mu.Unlock()
	fileMetadata := &msg
	c.files[msg.FileId] = fileMetadata
	log.Printf("File ID: %s\n", msg.FileId)
	// log.Printf("File Name: %s\n", msg.FileName)
	// log.Printf("File Extension: %s\n", msg.FileExtension)
	// log.Printf("File Size: %d\n", msg.FileSize)
	// log.Printf("Chunk Size: %d\n", msg.ChunkSize)
	// log.Println("Chunk Info:")
	// for chunkNumber, chunkMeta := range msg.ChunkInfo {
	// 	log.Printf("  Chunk Number: %d\n", chunkNumber)
	// 	log.Printf("    Chunk Hash: %s\n", chunkMeta.ChunkHash)
	// 	log.Printf("    Chunk Index: %d\n", chunkMeta.ChunkIndex)
	// 	log.Printf("    Peers With Chunk: %v\n", chunkMeta.PeersWithChunk)
	// }

	return nil
}

func (c *CentralServer) OnPeer(peer pkg.Peer, connType string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if connType == "central-server-peer" {
		c.peers[peer.RemoteAddr().String()] = peer
	} else {
		return nil
	}
	log.Printf("listener central server peer connected with  %s", peer.RemoteAddr())

	return nil

}
