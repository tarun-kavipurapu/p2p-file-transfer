package peer

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"tarun-kavipurapu/p2p-transfer/pkg"
	"time"
)

// this should have the Metadata of the files it posses
type PeerServer struct {
	peerLock           sync.Mutex
	peers              map[string]pkg.Peer //conn
	centralServerPeer  pkg.Peer
	fileMetaDataInfo   map[string]*pkg.FileMetaData //fileId-->metaData
	Transport          pkg.Transport
	peerServAddr       string
	cServerAddr        string
	store              Store
	completedDownloads map[string]bool
	downloadsMutex     sync.RWMutex
}

func init() {
	gob.Register(pkg.FileMetaData{})
	gob.Register(pkg.ChunkMetadata{})
	gob.Register(pkg.RequestChunkData{})
	gob.Register(pkg.ChunkRequestToPeer{})
	gob.Register(pkg.RegisterSeeder{})
}

func NewPeerServer(opts pkg.TransportOpts, cServerAddr string) *PeerServer {
	transport := pkg.NewTCPTransport(opts)
	peerServer := &PeerServer{
		peers:              make(map[string]pkg.Peer),
		peerServAddr:       transport.ListenAddr,
		peerLock:           sync.Mutex{},
		Transport:          transport,
		cServerAddr:        cServerAddr,
		store:              Store{},
		fileMetaDataInfo:   make(map[string]*pkg.FileMetaData),
		completedDownloads: make(map[string]bool),
		downloadsMutex:     sync.RWMutex{},
	}
	transport.OnPeer = peerServer.OnPeer

	log.Printf("[PeerServer] Initialized with address: %s", peerServer.peerServAddr)
	return peerServer
}

func (p *PeerServer) loop() {
	defer func() {
		log.Printf("[PeerServer] Stopping due to error or quit action")
		p.Transport.Close()
	}()
	log.Printf("[PeerServer] Starting main loop")
	for {
		select {
		case msg := <-p.Transport.Consume():
			dataMsg := new(pkg.DataMessage)
			buf := bytes.NewReader(msg.Payload)
			decoder := gob.NewDecoder(buf)
			if err := decoder.Decode(dataMsg); err != nil {
				log.Printf("[PeerServer] Failed to decode data: %v", err)
				continue
			}

			if err := p.handleMessage(msg.From, dataMsg); err != nil {
				log.Printf("[PeerServer] Error handling message from %s: %v", msg.From, err)
			}
		}
	}
}

func (p *PeerServer) handleMessage(from string, dataMsg *pkg.DataMessage) error {
	switch v := dataMsg.Payload.(type) {
	case pkg.FileMetaData:
		log.Printf("[PeerServer] Received FileMetaData for file: %s", v.FileName)
		return p.handleRequestChunks(from, v)
	case pkg.ChunkRequestToPeer:
		log.Printf("[PeerServer] Received ChunkRequestToPeer for file: %s, chunk: %d", v.FileId, v.ChunkId)
		return p.SendChunkData(from, v)
	default:
		return fmt.Errorf("unknown message type received from %s", from)
	}
}

func (p *PeerServer) SendChunkData(from string, v pkg.ChunkRequestToPeer) error {
	filePath := fmt.Sprintf("/mnt/d/Devlopement/p2p-file-transfer/chunks-%s/%s/%s", p.peerServAddr, v.FileId, v.ChunkName)
	log.Printf("[PeerServer] Sending chunk %d of file %s to %s", v.ChunkId, v.FileId, from)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open chunk file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	p.peerLock.Lock()
	peer := p.peers[from]
	p.peerLock.Unlock()

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int64(v.ChunkId))
	binary.Write(&buf, binary.LittleEndian, fileInfo.Size())

	if _, err := peer.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to send chunk metadata: %w", err)
	}

	n, err := io.Copy(peer, file)
	if err != nil {
		return fmt.Errorf("failed to send chunk data: %w", err)
	}

	log.Printf("[PeerServer] Sent %d bytes of chunk %d to %s", n, v.ChunkId, from)
	return nil
}

func (p *PeerServer) handleRequestChunks(from string, msg pkg.FileMetaData) error {
	log.Printf("[PeerServer] Handling request for chunks of file: %s", msg.FileName)
	err := p.handleChunks(msg)
	if err != nil {
		log.Printf("[PeerServer] Error handling chunks for file %s: %v", msg.FileName, err)
		return err
	}
	return nil
}
func (p *PeerServer) OnPeer(peer pkg.Peer, connType string) error {
	p.peerLock.Lock()
	defer p.peerLock.Unlock()

	switch connType {
	case "peer-server":
		p.peers[peer.RemoteAddr().String()] = peer
		log.Printf("[PeerServer] New peer connected: %s", peer.RemoteAddr())
	case "centralServer":
		p.centralServerPeer = peer
		log.Printf("[PeerServer] Connected to central server: %s", peer.RemoteAddr())
	default:
		log.Printf("[PeerServer] Peer connected as listener: %s", peer.RemoteAddr())
	}

	return nil
}

func (p *PeerServer) RegisterPeer() error {
	log.Printf("[PeerServer] Attempting to register with central server: %s", p.cServerAddr)
	err := p.Transport.Dial(p.cServerAddr, "centralServer")
	if err != nil {
		return fmt.Errorf("failed to dial central server: %w", err)
	}

	time.Sleep(1 * time.Second)
	p.peerLock.Lock()
	err = p.centralServerPeer.Send([]byte{pkg.IncomingMessage})
	p.peerLock.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send registration message: %w", err)
	}

	log.Printf("[PeerServer] Successfully registered with central server")
	return nil
}

func (p *PeerServer) RegisterFile(path string) error {
	log.Printf("[PeerServer] Registering file: %s", path)
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fileName := strings.Split(fileInfo.Name(), ".")
	hashString, err := pkg.HashFile(file)
	if err != nil {
		return fmt.Errorf("failed to hash file: %w", err)
	}

	dir := fmt.Sprintf("chunks-%s", p.peerServAddr)
	fileDirectory, err := p.store.createChunkDirectory(dir, hashString)
	if err != nil {
		return fmt.Errorf("failed to create chunk directory: %w", err)
	}

	const chunkSize = 1024 * 1024 * 4
	err, chunkMap := p.store.divideToChunk(file, chunkSize, fileDirectory, p.peerServAddr)
	if err != nil {
		return fmt.Errorf("failed to process chunks: %w", err)
	}

	dataMessage := pkg.DataMessage{
		Payload: pkg.FileMetaData{
			FileId:        hashString,
			FileName:      fileName[0],
			FileExtension: fileName[1],
			FileSize:      uint32(fileInfo.Size()),
			ChunkSize:     chunkSize,
			ChunkInfo:     chunkMap,
		},
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(dataMessage); err != nil {
		return fmt.Errorf("failed to encode file metadata: %w", err)
	}

	time.Sleep(time.Millisecond * 600)
	p.peerLock.Lock()
	err = p.centralServerPeer.Send(buf.Bytes())
	p.peerLock.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send file metadata to central server: %w", err)
	}

	log.Printf("[PeerServer] Successfully registered file: %s with ID: %s", fileInfo.Name(), hashString)
	return nil
}

func (p *PeerServer) registerAsSeeder(fileId string) error {
	p.downloadsMutex.RLock()
	isComplete, exists := p.completedDownloads[fileId]
	p.downloadsMutex.RUnlock()

	if !exists || !isComplete {
		return fmt.Errorf("file %s is not completely downloaded", fileId)
	}

	registerMsg := pkg.DataMessage{
		Payload: pkg.RegisterSeeder{
			FileId:   fileId,
			PeerAddr: p.peerServAddr,
		},
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(registerMsg); err != nil {
		return fmt.Errorf("failed to encode register seeder message: %w", err)
	}

	p.peerLock.Lock()
	err := p.centralServerPeer.Send(buf.Bytes())
	p.peerLock.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send register seeder message: %w", err)
	}

	log.Printf("[PeerServer] Registered as seeder for file: %s", fileId)
	return nil
}

func (p *PeerServer) RequestChunkData(fileId string) error {
	log.Printf("[PeerServer] Requesting chunk data for file: %s", fileId)
	dataMessage := pkg.DataMessage{
		Payload: pkg.RequestChunkData{
			FileId: fileId,
		},
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(dataMessage); err != nil {
		return fmt.Errorf("failed to encode chunk data request: %w", err)
	}

	p.peerLock.Lock()
	err := p.centralServerPeer.Send(buf.Bytes())
	p.peerLock.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send chunk data request: %w", err)
	}

	log.Printf("[PeerServer] Successfully sent request for chunk data of file: %s", fileId)
	return nil
}

func (p *PeerServer) Start() error {
	log.Printf("[PeerServer] Starting peer server on address: %s", p.Transport.Addr())

	err := p.Transport.ListenAndAccept("peer-server")
	if err != nil {
		return fmt.Errorf("failed to start listening: %w", err)
	}

	err = p.RegisterPeer()
	if err != nil {
		log.Printf("[PeerServer] Warning: Failed to register peer: %v", err)
	}

	p.loop()
	return nil
}
