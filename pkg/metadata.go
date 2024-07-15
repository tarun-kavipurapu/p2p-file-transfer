package pkg

import "time"

type PeerMetadata struct {
	PeerId     string
	Addr       string
	Port       uint16
	LastActive time.Time
}

type ChunkMetadata struct {
	ChunkHash      string
	ChunkIndex     uint32
	PeersWithChunk []string //here peers adress should be server adress of the peer
}
type FileMetaData struct {
	FileId        string //this is nothing but file hash
	FileName      string
	FileExtension string
	FileSize      uint32
	ChunkSize     uint32
	ChunkInfo     map[uint32]*ChunkMetadata //chunkNumber-->ChunkMetaData

}
