package peer

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"tarun-kavipurapu/stream-files/pkg"
)

const (
	INFO  = "\033[1;34m%s\033[0m" // Blue
	WARN  = "\033[1;33m%s\033[0m" // Yellow
	ERROR = "\033[1;31m%s\033[0m" // Red
)

type PeerServer struct {
	mu              sync.Mutex                //to protect peerCOnnections
	peerConnections map[string]*pkg.TCPClient // Connections to other peers

	tcpTransport      *pkg.TCPTransport
	centralServerConn *pkg.TCPClient
	peerId            string //this will be the ip aress and the port in which the peer will start listening
	fileMetaData      map[string]*pkg.FileMetaData
	wg                sync.WaitGroup
}

func NewPeerServer(listenerAddr string, decoder pkg.Decoder) *PeerServer {
	return &PeerServer{
		tcpTransport:    pkg.NewTCPTransport(listenerAddr, decoder),
		peerConnections: make(map[string]*pkg.TCPClient),
		peerId:          listenerAddr,
		fileMetaData:    make(map[string]*pkg.FileMetaData),
	}
}
func (p *PeerServer) CloseStream() {
	p.wg.Wait()
}
func (p *PeerServer) StartListener() error {
	p.wg.Add(2)

	defer p.wg.Done()
	go func() {
		if err := p.tcpTransport.Start(); err != nil {
			log.Fatalf("Failed to start TCP server: %v", err)
		}
	}()
	go p.consumeMessages()
	return nil

}

func (p *PeerServer) consumeMessages() {

	for message := range p.tcpTransport.Consume() {
		switch message.Type {

		case "response_file":
			var fileMetaData pkg.FileMetaData
			p.tcpTransport.Decoder.DecodeFromBytes(message.Payload, &fileMetaData)
			p.ResponseFile(fileMetaData)
		}
	}
}

// Connect establishes a connection to the central server
func (p *PeerServer) ConnectServer(addr string) error {
	conn, err := p.tcpTransport.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	p.centralServerConn = pkg.NewTCPClient(conn, true)
	// p.peerId = conn.LocalAddr().String()
	log.Printf(INFO, "Connected to central server: %s", addr)
	return nil
}

func (p *PeerServer) SendMessage(message pkg.Message) error {
	data, err := p.tcpTransport.Decoder.EncodeToBytes(&message)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	err = p.centralServerConn.Send(data.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	log.Printf(INFO, "Message sent: %s", message.Type)
	return nil
}

func (p *PeerServer) RegisterPeer() error {
	message := pkg.Message{
		Type:     "register_peer",
		From:     "peer",
		FromAddr: p.peerId,
	}
	err := p.SendMessage(message)
	if err != nil {
		return fmt.Errorf("failed to register peer: %w", err)
	}
	log.Printf(INFO, "Peer registered successfully")
	return nil
}

func (p *PeerServer) RegisterFile() {
	store := &Store{}
	// log.Printf(ERROR, fmt.Sprintf("%s"), p.peerId)
	file, err := store.openFile("/mnt/d/Devlopement/go-network-Stream/TestingFiles/test_image.tif")
	if err != nil {
		log.Fatalf(ERROR, fmt.Sprintf("Failed to open file: %v", err))
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf(ERROR, fmt.Sprintf("Failed to get file info: %v", err))
	}
	fileName := strings.Split(fileInfo.Name(), ".")
	extensionName := fileName[1]

	hashString, err := pkg.HashFile(file)
	if err != nil {
		log.Fatalf(ERROR, fmt.Sprintf("Failed to hash file: %v", err))
	}

	log.Printf(INFO, fmt.Sprintf("File hash: %s", hashString))

	fileDirectory, err := store.createChunkDirectory(hashString)
	if err != nil {
		log.Fatalf(ERROR, fmt.Sprintf("Failed to create chunk directory: %v", err))
	}

	const chunkSize = 1024 * 1024 * 4
	err, chunkMap := store.divideToChunk(file, chunkSize, fileDirectory, p.peerId)
	if err != nil {
		log.Fatalf(ERROR, fmt.Sprintf("Failed to process chunks: %v", err))
	}

	fileMetaData := pkg.FileMetaData{
		FileId:        hashString,
		FileName:      fileName[0],
		FileExtension: extensionName,
		FileSize:      uint32(fileInfo.Size()),
		ChunkSize:     chunkSize,
		ChunkInfo:     chunkMap,
	}

	data, err := p.tcpTransport.Decoder.EncodeToBytes(&fileMetaData)
	if err != nil {
		log.Fatalf(ERROR, fmt.Sprintf("Failed to encode file metadata: %v", err))
	}

	message := pkg.Message{
		Type:     "register_file",
		From:     "peer",
		FromAddr: p.peerId,
		Payload:  data.Bytes(),
	}

	err = p.SendMessage(message)
	if err != nil {
		log.Fatalf(ERROR, fmt.Sprintf("Failed to send message: %v", err))
	}
	log.Printf(INFO, "File registered successfully")
}

func (p *PeerServer) RequestFile(fileId string) error {
	log.Println("Request File running...")

	data, err := p.tcpTransport.Decoder.EncodeToBytes(fileId)

	message := pkg.Message{
		Type:     "request_file",
		From:     "peer",
		FromAddr: p.peerId,
		Payload:  data.Bytes(),
	}
	// log.Println((message.Payload))
	err = p.SendMessage(message)
	if err != nil {
		return fmt.Errorf("Failed To send Request to Central Server")
	}
	return nil
}
func (p *PeerServer) ResponseFile(fileMetaData pkg.FileMetaData) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fileId := fileMetaData.FileId
	p.fileMetaData[fileId] = &fileMetaData
	log.Println("I am here")
	log.Println(p.fileMetaData)
	myMap := p.fileMetaData[fileId].ChunkInfo
	for key, val := range myMap {
		fmt.Printf("Key: %d, Value: %s\n", key, val)
	}

}
