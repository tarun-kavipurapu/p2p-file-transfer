package peer

import (
	"bytes"
	"fmt"
	"log"
	"tarun-kavipurapu/stream-files/pkg"
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

func (p *PeerServer) handleChunks(fileId string) {
	log.Println("Entering handleChunks") // Added log

	// p.mu.Lock()
	log.Println("Acquired lock in handleChunks") // Added log
	fileMetaData := p.fileMetaData[fileId]
	// p.mu.Unlock()

	log.Println("Released lock in handleChunks") // Added log
	log.Println(ERROR, "Hi, I am here")          // Added log

	if fileMetaData == nil {
		log.Printf(ERROR, "File metadata not found for fileId: %s", fileId)
		return
	}

	var chunkStatus []ChunkStatus
	numberOfchunks := uint32(len(fileMetaData.ChunkInfo))
	for i := uint32(1); i < numberOfchunks; i++ {
		chunkStatus = append(chunkStatus, ChunkStatus{Index: i})
	}

	chunkPeerAssign := p.assignChunks(fileId, chunkStatus)
	log.Println(chunkPeerAssign, "Chunks peer assign")

	for chunkIndex, peerAddr := range chunkPeerAssign {
		p.wg.Add(1)
		go p.fetchData(chunkIndex, peerAddr, fileId)
	}

	log.Println("Waiting for all fetchData goroutines to complete") // Added log
	p.wg.Wait()
	log.Println("Completed handleChunks execution") // Added log
}

func (p *PeerServer) assignChunks(fileId string, chunkStatus []ChunkStatus) map[uint32]string {
	//we need to perform the load balancing which means that i need to select the ip adress to which least amount of chunks are assigned
	chunkPeerAssign := make(map[uint32]string) //chunkIndex->peer
	peerLoad := make(map[string]uint32)        //peerId ->number of chunks
	file := p.fileMetaData[fileId]
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

// This request is being sent to the peer{
type ChunkRequest struct {
	FileId     string
	ChunkIndex uint32
	Addr       string //peer adress which have the chunk
}

func (p *PeerServer) fetchData(chunkIndex uint32, peerAddr string, fileId string) {
	//connect
	//send the Get Chunk message with the CHunkRequest then we will design a response mechanism
	defer p.wg.Done()

	err := p.Connect(peerAddr, "peer")
	if err != nil {
		log.Fatalln(err)
	}
	chunkRequest := ChunkRequest{
		FileId:     fileId,
		ChunkIndex: chunkIndex,
		Addr:       peerAddr,
	}
	data, err := p.tcpTransport.Decoder.EncodeToBytes(chunkRequest)
	if err != nil {
		log.Fatalln(err)
	}
	message := pkg.Message{
		Type:     "request_chunk",
		From:     "peer",
		FromAddr: p.peerId,
		Payload:  data.Bytes(),
	}
	err = p.SendMessage(message, p.peerConnections[peerAddr])
	if err != nil {
		log.Fatalln(err)
	}

}
func (p *PeerServer) handleChunkResult(chunkResult ChunkResult) error {
	defer p.wg.Done()
	if chunkResult.Err != nil {
		return fmt.Errorf(ERROR, "Error fetching chunk %d: %v", chunkResult.Index, chunkResult.Err)

	}

	fileDirectory, err := p.store.createChunkDirectory("chunks-peer1", chunkResult.FileId)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(chunkResult.Data)
	err = p.store.writeChunk(fileDirectory, chunkResult.Index, buffer)
	if err != nil {
		return err
	}

	return nil
}
