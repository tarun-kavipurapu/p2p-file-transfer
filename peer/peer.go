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
	peerLock          sync.Mutex
	peers             map[string]pkg.Peer //conn
	centralServerPeer pkg.Peer
	fileMetaDataInfo  map[string]*pkg.FileMetaData //fileId-->metaData
	Transport         pkg.Transport
	peerServAddr      string
	cServerAddr       string
	store             Store
}

func init() {
	gob.Register(pkg.FileMetaData{})
	gob.Register(pkg.ChunkMetadata{})
	gob.Register(pkg.RequestChunkData{})
	gob.Register(pkg.ChunkRequestToPeer{})
}

func NewPeerServer(opts pkg.TransportOpts, cServerAddr string) *PeerServer {
	transport := pkg.NewTCPTransport(opts)
	peerServer := &PeerServer{
		peers:            make(map[string]pkg.Peer),
		peerServAddr:     transport.ListenAddr,
		peerLock:         sync.Mutex{},
		Transport:        transport,
		cServerAddr:      cServerAddr,
		store:            Store{},
		fileMetaDataInfo: make(map[string]*pkg.FileMetaData),
	}
	transport.OnPeer = peerServer.OnPeer

	return peerServer
}

func (p *PeerServer) loop() {
	defer func() {
		log.Println("Central Server has stopped due to error or quit action")
		p.Transport.Close()
	}()
	for {
		select {
		case msg := <-p.Transport.Consume():
			// log.Println((msg.Payload))
			dataMsg := new(pkg.DataMessage)
			buf := bytes.NewReader(msg.Payload)
			// log.Println(buf)
			decoder := gob.NewDecoder(buf)
			if err := decoder.Decode(dataMsg); err != nil {
				log.Println("failed to decode data: %w", err)
			}

			if err := p.handleMessage(msg.From, dataMsg); err != nil {
				log.Println("Error handling Mesage:", err)
			}

		}
	}
}
func (p *PeerServer) handleMessage(from string, dataMsg *pkg.DataMessage) error {
	switch v := dataMsg.Payload.(type) {
	case pkg.FileMetaData:
		{
			return p.handleRequestChunks(from, v)
		}
	case pkg.ChunkRequestToPeer:
		{
			return p.SendChunkData(from, v)
		}
	}

	return nil
}
func (p *PeerServer) SendChunkData(from string, v pkg.ChunkRequestToPeer) error {
	filePath := fmt.Sprintf("/mnt/d/Devlopement/p2p-file-transfer/chunks/%s/%s", v.FileId, v.ChunkName)
	log.Println("Chunk Id ", v.ChunkId)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	fileInfo, err := file.Stat()

	p.peerLock.Lock()
	peer := p.peers[from]
	p.peerLock.Unlock()

	// peer.Send([]byte{IncomingStream})
	var buf bytes.Buffer

	binary.Write(&buf, binary.LittleEndian, int64(v.ChunkId))
	binary.Write(&buf, binary.LittleEndian, fileInfo.Size())

	if _, err := peer.Write(buf.Bytes()); err != nil {
		return err
	}
	n, err := io.Copy(peer, file)
	if err != nil {
		return err
	}
	fmt.Printf("[%s] written (%d) bytes over the network to %s with chunk id (%d)\n", p.Transport.Addr(), n, from, v.ChunkId)

	return nil
}

func (p *PeerServer) handleRequestChunks(from string, msg pkg.FileMetaData) error {
	// from  the message get the which chunks is with which peers and assign peers to chunks and from that assignment fetch the
	log.Println(msg)

	for index, ele := range msg.ChunkInfo {
		log.Println(index, ele)
	}

	//
	p.handleChunks(msg)
	return nil
}
func (p *PeerServer) OnPeer(peer pkg.Peer, connType string) error {
	p.peerLock.Lock()
	defer p.peerLock.Unlock()

	if connType == "peer-server" {
		p.peers[peer.RemoteAddr().String()] = peer
	} else if connType == "centralServer" {
		p.centralServerPeer = peer
	} else {
		log.Printf("peer connected as a listener so do nothing")
	}

	log.Printf("connected with remote %s", peer.RemoteAddr())
	return nil
}

func (p *PeerServer) RegisterPeer() error {
	err := p.Transport.Dial(p.cServerAddr, "centralServer")
	if err != nil {
		return nil
	}
	time.Sleep(1 * time.Second)
	p.peerLock.Lock()
	err = p.centralServerPeer.Send([]byte{pkg.IncomingMessage})
	p.peerLock.Unlock()
	return err
}

func (p *PeerServer) RegisterFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}
	fileName := strings.Split(fileInfo.Name(), ".")
	hashString, err := pkg.HashFile(file)
	if err != nil {
		log.Fatalf("Failed to hash file: %v", err)
	}
	fileDirectory, err := p.store.createChunkDirectory("chunks", hashString)
	if err != nil {
		log.Fatalf("Failed to create chunk directory: %v", err)
	}

	const chunkSize = 1024 * 1024 * 4
	err, chunkMap := p.store.divideToChunk(file, chunkSize, fileDirectory, p.peerServAddr)
	if err != nil {
		log.Fatalf("Failed to process chunks: %v", err)
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
		return fmt.Errorf("failed to encode data: %w", err)
	}

	if err != nil {
		return err
	}
	time.Sleep(time.Millisecond * 600)
	p.peerLock.Lock()
	err = p.centralServerPeer.Send(buf.Bytes())
	if err != nil {
		return err
	}
	p.peerLock.Unlock()

	return nil
}

func (p *PeerServer) RequestChunkData(fileId string) error {
	dataMessage := pkg.DataMessage{
		Payload: pkg.RequestChunkData{
			FileId: fileId,
		},
	}
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(dataMessage); err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}

	p.peerLock.Lock()
	err := p.centralServerPeer.Send(buf.Bytes())
	if err != nil {
		return err
	}
	p.peerLock.Unlock()

	return nil
}

func (p *PeerServer) Start() error {
	fmt.Printf("[%s] starting Peer ...\n", p.Transport.Addr())

	err := p.Transport.ListenAndAccept("peer-server")
	if err != nil {
		return err
	}
	//try to register the peer to the central server
	err = p.RegisterPeer()
	if err != nil {
		log.Println("Error Register peer", err)
	}
	p.loop()
	return nil
}
