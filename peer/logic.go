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

type ChunkJob struct {
	FileId     string
	ChunkIndex uint32
	PeerAddr   string
	ChunkHash  string
	p          *PeerServer
}

func (cj *ChunkJob) Execute() error {
	return cj.p.fileRequest(cj.FileId, cj.ChunkIndex, cj.PeerAddr, cj.ChunkHash)
}

func (p *PeerServer) handleChunks(fileMetaData pkg.FileMetaData) error {
	numOfChunks := uint32(len(fileMetaData.ChunkInfo))
	chunkPeerAssign := assignChunks(fileMetaData)

	log.Printf("[PeerServer] Starting download of file %s (%s) with %d chunks", fileMetaData.FileName, fileMetaData.FileId, numOfChunks)

	const maxWorkers = 5
	workerPool := NewWorkerPool(maxWorkers)
	workerPool.Start()

	var wg sync.WaitGroup
	for index, peerAddr := range chunkPeerAssign {
		wg.Add(1)
		go func(index uint32, peerAddr string) {
			defer wg.Done()
			chunk := fileMetaData.ChunkInfo[index]
			job := &ChunkJob{
				FileId:     fileMetaData.FileId,
				ChunkIndex: index,
				PeerAddr:   peerAddr,
				ChunkHash:  chunk.ChunkHash,
				p:          p,
			}
			workerPool.Submit(job)
		}(index, peerAddr)
	}
	go func() {
		wg.Wait()
		workerPool.Stop()
	}()

	successfulChunks := 0
	var resultLock sync.Mutex
	var lastError error

	for result := range workerPool.Results() {
		chunkJob := result.Job.(*ChunkJob)
		if result.Err != nil {
			log.Printf("[PeerServer] Failed to fetch chunk %d from peer %s: %v", chunkJob.ChunkIndex, chunkJob.PeerAddr, result.Err)
			lastError = result.Err
		} else {
			resultLock.Lock()
			successfulChunks++
			resultLock.Unlock()
			log.Printf("[PeerServer] Fetched chunk %d/%d from peer %s", successfulChunks, numOfChunks, chunkJob.PeerAddr)
		}
	}

	<-workerPool.Done()

	if successfulChunks != int(numOfChunks) {
		return fmt.Errorf("download incomplete for file %s. Fetched %d/%d chunks. Last error: %v", fileMetaData.FileName, successfulChunks, numOfChunks, lastError)
	}

	log.Printf("[PeerServer] All %d chunks fetched successfully for file %s. Reassembling...", numOfChunks, fileMetaData.FileName)
	err := p.store.ReassembleFile(fileMetaData.FileId, fileMetaData.FileName, fileMetaData.FileExtension, p.peerServAddr)
	if err != nil {
		return fmt.Errorf("error reassembling file %s: %w", fileMetaData.FileName, err)
	}

	log.Printf("[PeerServer] File %s reassembled successfully", fileMetaData.FileName)

	p.downloadsMutex.Lock()
	p.completedDownloads[fileMetaData.FileId] = true
	p.downloadsMutex.Unlock()

	log.Printf("[PeerServer] Marked file %s (%s) as completed download", fileMetaData.FileName, fileMetaData.FileId)

	err = p.registerAsSeeder(fileMetaData.FileId)
	if err != nil {
		return fmt.Errorf("failed to register as seeder for file %s: %w", fileMetaData.FileName, err)
	}

	log.Printf("[PeerServer] Successfully registered as a new seeder for file %s", fileMetaData.FileName)
	return nil
}
func (p *PeerServer) fileRequest(fileId string, chunkIndex uint32, peerAddr string, chunkHash string) error {
	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", peerAddr, err)
	}
	defer conn.Close()

	peer := pkg.NewTCPPeer(conn, true)

	// Update peer map
	p.peerLock.Lock()
	p.peers[peerAddr] = peer
	p.peerLock.Unlock()

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
		return fmt.Errorf("failed to encode chunk request: %w", err)
	}

	if err := peer.Send(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to send chunk request to peer %s: %w", peerAddr, err)
	}

	var chunkId int64
	var fileSize int64

	if err := binary.Read(peer, binary.LittleEndian, &chunkId); err != nil {
		return fmt.Errorf("failed to read chunk id from peer %s: %w", peerAddr, err)
	}

	if err := binary.Read(peer, binary.LittleEndian, &fileSize); err != nil {
		return fmt.Errorf("failed to read file size from peer %s: %w", peerAddr, err)
	}

	baseDir := fmt.Sprintf("chunks-%s", p.peerServAddr)
	folderDirectory, err := p.store.createChunkDirectory(baseDir, fileId)
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
		return fmt.Errorf("failed to copy chunk data from peer %s: %w", peerAddr, err)
	}

	if _, err := writeFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start of file %s: %w", filePath, err)
	}

	if err := pkg.CheckFileHash(writeFile, chunkHash); err != nil {
		return fmt.Errorf("chunk hash verification failed: %w", err)
	}

	return nil
}
func assignChunks(fileMetaData pkg.FileMetaData) map[uint32]string {
	chunkPeerAssign := make(map[uint32]string) //chunkIndex->peer
	peerLoad := make(map[string]uint32)        //peerId ->number of chunks

	for index, chunkInfo := range fileMetaData.ChunkInfo {
		peers := chunkInfo.PeersWithChunk
		if len(peers) > 0 {
			minPeer := peers[0]
			for _, peer := range peers {
				if peerLoad[peer] < peerLoad[minPeer] {
					minPeer = peer
				}
			}
			peerLoad[minPeer]++
			chunkPeerAssign[index] = minPeer
		}
	}

	return chunkPeerAssign
}
