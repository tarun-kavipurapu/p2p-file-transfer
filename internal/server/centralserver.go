package server

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"tarun-kavipurapu/stream-files/pkg"
	"time"
)

type CentralServer struct {
	tcpClient    *pkg.TCPClient    //this is the client component that interacts with peers
	tcpTransport *pkg.TCPTransport //this is the server component that interacts with the peers
	mu           sync.Mutex
	peers        map[string]*pkg.PeerMetadata
	files        map[string]*pkg.FileMetaData
	serverId     string
	wg           sync.WaitGroup
}

func NewCentralServer(listenerAddr string, decoder pkg.Decoder) *CentralServer {
	return &CentralServer{
		tcpTransport: pkg.NewTCPTransport(listenerAddr, decoder),
		peers:        make(map[string]*pkg.PeerMetadata),
		files:        make(map[string]*pkg.FileMetaData),
		serverId:     listenerAddr,
	}
}

func (cs *CentralServer) CloseStream() {
	cs.wg.Wait()
}

func (cs *CentralServer) Start() error {
	cs.wg.Add(2)
	go func() {
		defer cs.wg.Done()
		if err := cs.tcpTransport.Start(); err != nil {
			log.Fatalf("Failed to start TCP server: %v", err)
		}
	}()
	go cs.consumeMessages()
	return nil

}
func (cs *CentralServer) ConnectPeer(addr string) error {
	conn, err := cs.tcpTransport.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	cs.tcpClient = pkg.NewTCPClient(conn, true)
	// p.peerId = conn.LocalAddr().String()
	log.Printf("Connected to peer server: %s", addr)
	return nil
}
func (cs *CentralServer) SendResponse(addr string, message pkg.Message) error {
	err := cs.ConnectPeer(addr) //connect tot that specific peer first
	if err != nil {
		return err
	}
	data, err := cs.tcpTransport.Decoder.EncodeToBytes(&message)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	err = cs.tcpClient.Send(data.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	log.Printf("Message sent: %s", message.Type)
	return nil
}

func (cs *CentralServer) consumeMessages() {

	for {
		select {
		case message := <-cs.tcpTransport.Consume():
			switch message.Type {
			case "register_peer":
				log.Printf("Registering peer: %s\n", message.From)
				if err := cs.registerPeer(message); err != nil {
					log.Printf("Error registering peer: %v", err)
				} else {
					log.Printf("Registration done: %s\n", message.From)
				}
			case "register_file":
				//Decode the meta Data fromm the input
				var fileMetaData pkg.FileMetaData
				err := cs.tcpTransport.Decoder.DecodeFromBytes(message.Payload, &fileMetaData)
				if err != nil {
					log.Printf("Error decoding file metadata: %v", err)
					continue
				}

				// myMap := fileMetaData.ChunkInfo
				// for key, val := range myMap {
				// 	fmt.Printf("Key: %d, Value: %s\n", key, val)
				// }

				log.Printf("Registering file from peer: %s\n", message.From)
				if err := cs.registerFile(fileMetaData, message.FromAddr); err != nil {
					log.Printf("Error registering file: %v", err)
				} else {
					log.Printf("File registration done: %s\n", message.From)
				}
			case "request_file":
				log.Println("I am here")
				var fileID string
				err := cs.tcpTransport.Decoder.DecodeFromBytes(message.Payload, &fileID)
				log.Println("File ID received", (fileID))
				if err != nil {
					log.Printf("Error decoding file Id: %v", err)
					continue
				}

				cs.requestFile(fileID, message.FromAddr)

			case "request_chunk":
				log.Printf("Peer %s requesting chunk\n", message.From)
			default:
				log.Printf("Unknown message type: %s\n", message.Type)
			}
		}
	}
}

func (cs *CentralServer) registerPeer(message pkg.Message) error {
	temp := strings.Split(message.FromAddr, ":")
	if len(temp) != 2 {
		return fmt.Errorf("invalid peer address: %s", message.FromAddr)
	}

	ip := temp[0]
	portStr := temp[1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %v", err)
	}

	peerMeta := &pkg.PeerMetadata{
		PeerId:     message.FromAddr,
		Addr:       ip,
		Port:       uint16(port),
		LastActive: time.Now(),
	}

	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.peers[message.FromAddr] = peerMeta
	return nil
}

func (cs *CentralServer) registerFile(fileMetaData pkg.FileMetaData, peerId string) error {
	cs.mu.Lock()

	defer cs.mu.Unlock()

	fileId := fileMetaData.FileId
	cs.files[fileId] = &fileMetaData

	for _, chunkMetaData := range fileMetaData.ChunkInfo {
		chunkMetaData.PeersWithChunk = append(chunkMetaData.PeersWithChunk, peerId)
	}
	// for _, chunkMetaData := range fileMetaData.ChunkInfo {
	// 	fmt.Println(chunkMetaData)
	// }
	return nil
}
func (cs *CentralServer) requestFile(fileId string, peerId string) error {
	cs.mu.Lock()

	defer cs.mu.Unlock()
	fileMap, ok := cs.files[fileId]
	if !ok {
		return fmt.Errorf("File not present in the network please register the file ")
	}
	data, err := cs.tcpTransport.Decoder.EncodeToBytes(fileMap)
	if err != nil {
		return err
	}
	//sending response
	message := pkg.Message{
		Type:     "response_file",
		From:     "central",
		FromAddr: cs.serverId,
		Payload:  data.Bytes(),
	}
	err = cs.SendResponse(peerId, message)
	if err != nil {
		return err
	}

	return nil
}
