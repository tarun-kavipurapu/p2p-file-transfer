package peer

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"tarun-kavipurapu/p2p-transfer/pkg"
)

type ChunkStatus struct {
	Index      uint32
	isFetched  bool
	isFetching bool
}

type ChunkResult struct {
	FileId string
	Index  uint32
	Data   []byte
	Err    error
}

func (p *PeerServer) handleChunks(fileMetaData pkg.FileMetaData) {

	var chunkStatus []ChunkStatus

	numOfChunks := uint32(len(fileMetaData.ChunkInfo))

	for i := uint32(1); i <= numOfChunks; i++ {
		chunkStatus = append(chunkStatus, ChunkStatus{Index: i})
	}
	chunkPeerAssign := assignChunks(fileMetaData, chunkStatus)
	log.Println(chunkPeerAssign)
	var wg sync.WaitGroup
	for index, ele := range chunkPeerAssign {
		chunk := fileMetaData.ChunkInfo[index]
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.fileRequest(fileMetaData.FileId, index, ele, chunk.ChunkHash)

		}()
	}
	wg.Wait()
}
func (p *PeerServer) fileRequest(fileId string, chunkIndex uint32, peerAddr string, chunkHash string) error {
	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %w", peerAddr, err)
	}
	defer conn.Close()

	peer := pkg.NewTCPPeer(conn, true)

	// Update peer map
	p.peerLock.Lock()
	p.peers[peerAddr] = peer
	p.peerLock.Unlock()

	// Ensure peer is removed from map and connection is closed when we're done
	defer func() {
		p.peerLock.Lock()
		delete(p.peers, peerAddr)
		p.peerLock.Unlock()
	}()

	dataMessage := pkg.DataMessage{
		Payload: pkg.ChunkRequestToPeer{
			FileId:    fileId,
			ChunkHash: chunkHash,
			ChunkId:   chunkIndex,
			ChunkName: fmt.Sprintf("chunk_%d.chunk", chunkIndex),
		},
	}
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(dataMessage); err != nil {
		return fmt.Errorf("failed to encode data message: %w", err)
	}

	if err := peer.Send(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to send data message to peer %s: %w", peerAddr, err)
	}

	// Handle response
	var chunkId int64
	var fileSize int64

	if err := binary.Read(peer, binary.LittleEndian, &chunkId); err != nil {
		return fmt.Errorf("failed to read chunk id from peer %s: %w", peerAddr, err)
	}

	if err := binary.Read(peer, binary.LittleEndian, &fileSize); err != nil {
		return fmt.Errorf("failed to read file size from peer %s: %w", peerAddr, err)
	}

	log.Printf("Receiving chunk id: %d, file size: %d\n", chunkId, fileSize)

	folderDirectory, err := p.store.createChunkDirectory("chunks-new", fileId)
	if err != nil {
		return fmt.Errorf("failed to create chunk directory: %w", err)
	}

	chunkName := fmt.Sprintf("chunk_%d.chunk", chunkIndex)
	filePath := filepath.Join(folderDirectory, chunkName)

	writeFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer writeFile.Close()

	if _, err := io.CopyN(writeFile, peer, fileSize); err != nil {
		return fmt.Errorf("failed to copy file from peer %s: %w", peerAddr, err)
	}

	// Re-open the file to hash it
	if _, err := writeFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start of file %s: %w", filePath, err)
	}

	//check the validity of the file with the hash
	// Check the hash of the file
	if err := pkg.CheckFileHash(writeFile, chunkHash); err != nil {
		return err
	}

	log.Printf("Successfully received and saved chunk_%d.chunk\n", chunkId)

	return nil
}
func assignChunks(fileMetaData pkg.FileMetaData, chunkStatus []ChunkStatus) map[uint32]string {
	//we need to perform the load balancing which means that i need to select the ip adress to which least amount of chunks are assigned
	//in future we need make this ee=vene more solid by making it request another peer if the peer over here is not onnline
	chunkPeerAssign := make(map[uint32]string) //chunkIndex->peer
	peerLoad := make(map[string]uint32)        //peerId ->number of chunks
	file := fileMetaData
	for _, chunk := range chunkStatus {
		if !chunk.isFetched && !chunk.isFetching {

			peers := file.ChunkInfo[chunk.Index].PeersWithChunk //select the peers array
			if len(peers) > 0 {
				minPeer := peers[0]
				for _, peer := range peers {
					if peerLoad[peer] < peerLoad[minPeer] {
						minPeer = peer
					}

				}
				peerLoad[minPeer]++
				chunkPeerAssign[chunk.Index] = minPeer
				chunk.isFetching = true

			}
		}

	}
	//i will traverse through the array of peers for a specific chunk  and enter them into the peerLoad map select the peer which has the minimum load
	return chunkPeerAssign
}
