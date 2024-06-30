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
	chunkResult       chan ChunkResult
	store             *Store
}

func NewPeerServer(listenerAddr string, decoder pkg.Decoder) *PeerServer {
	return &PeerServer{
		tcpTransport:    pkg.NewTCPTransport(listenerAddr, decoder),
		peerConnections: make(map[string]*pkg.TCPClient),
		peerId:          listenerAddr,
		fileMetaData:    make(map[string]*pkg.FileMetaData),
		chunkResult:     make(chan ChunkResult),
		store:           &Store{},
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

	for {
		select {
		case message := <-p.tcpTransport.Consume():
			switch message.Type {
			case "response_file":
				var fileMetaData pkg.FileMetaData
				p.tcpTransport.Decoder.DecodeFromBytes(message.Payload, &fileMetaData)
				p.ResponseFile(fileMetaData)
			case "request_chunk":
				var chunkRequest ChunkRequest
				p.tcpTransport.Decoder.DecodeFromBytes(message.Payload, &chunkRequest)
				p.ResponseChunk(chunkRequest)
			case "response_chunk":
				var chunkResult ChunkResult
				p.tcpTransport.Decoder.DecodeFromBytes(message.Payload, &chunkResult)

				p.chunkResult <- chunkResult
			}

		case chunkResult := <-p.chunkResult:
			p.wg.Add(1)
			go p.handleChunkResult(chunkResult)
			p.wg.Wait()

		}

	}
}

func (p *PeerServer) Connect(addr string, connectionType string) error {
	conn, err := p.tcpTransport.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	switch connectionType {
	case "central":
		p.centralServerConn = pkg.NewTCPClient(conn, true)
		log.Printf(INFO, "Connected to central server: %s", addr)
	case "peer":
		p.peerConnections[addr] = pkg.NewTCPClient(conn, true)
		log.Printf(INFO, "Connected to peer server: %s", addr)
	default:
		return fmt.Errorf("unknown connection type: %s", connectionType)
	}

	return nil
}

//tcpClient is a client component which may signify client component of both peer and central server

func (p *PeerServer) SendMessage(message pkg.Message, tcpClient *pkg.TCPClient) error {
	data, err := p.tcpTransport.Decoder.EncodeToBytes(&message)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	err = tcpClient.Send(data.Bytes())
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
	err := p.SendMessage(message, p.centralServerConn)
	if err != nil {
		return fmt.Errorf("failed to register peer: %w", err)
	}
	log.Printf(INFO, "Peer registered successfully")
	return nil
}

func (p *PeerServer) RegisterFile() {
	// store := &Store{}
	// log.Printf(ERROR, fmt.Sprintf("%s"), p.peerId)
	file, err := p.store.openFile("/mnt/d/Devlopement/go-network-Stream/TestingFiles/test_image.tif")
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

	fileDirectory, err := p.store.createChunkDirectory("chunks", hashString)
	if err != nil {
		log.Fatalf(ERROR, fmt.Sprintf("Failed to create chunk directory: %v", err))
	}

	const chunkSize = 1024 * 1024 * 4
	err, chunkMap := p.store.divideToChunk(file, chunkSize, fileDirectory, p.peerId)
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

	err = p.SendMessage(message, p.centralServerConn)
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
	err = p.SendMessage(message, p.centralServerConn)
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
	p.handleChunks(fileId)
}

// i am responding to the peer which has requested a specifc chunk
func (p *PeerServer) ResponseChunk(chunkRequest ChunkRequest) {
	chunkIndex := chunkRequest.ChunkIndex
	sendToAddr := chunkRequest.Addr
	fileId := chunkRequest.FileId
	//fetch the file with this particular chunk index
	p.store.fetchFileByChunkId(fileId, chunkIndex)

	data, err := p.store.fetchFileByChunkId(fileId, chunkIndex)
	chunkResponse := ChunkResult{
		FileId: fileId,
		Index:  chunkIndex,
		Data:   data,
		Err:    err,
	}
	bytesData, err := p.tcpTransport.Decoder.EncodeToBytes(chunkResponse)
	message := pkg.Message{
		Type:     "response_chunk",
		From:     "peer",
		FromAddr: p.peerId,
		Payload:  bytesData.Bytes(),
	}
	err = p.Connect(sendToAddr, "peer")
	if err != nil {
		log.Printf(ERROR, "Failed to connect to requesting peer: %s, error: %v", sendToAddr, err)
		return
	}
	peerClient := p.peerConnections[sendToAddr]
	err = p.SendMessage(message, peerClient)
	if err != nil {
		log.Printf(ERROR, "Failed to send chunk response: %v", err)
	}
	peerClient.Conn.Close()
	delete(p.peerConnections, sendToAddr)
	//this data is the data in that file

}
