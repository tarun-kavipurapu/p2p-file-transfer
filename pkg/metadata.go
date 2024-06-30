package pkg

import "time"

type PeerMetadata struct {
	PeerId     string
	Addr       string
	Port       uint16
	LastActive time.Time
}

type ChunkMetadata struct {
	ChunkIndex     uint32
	PeersWithChunk []string
}
type FileMetaData struct {
	FileId        string
	FileName      string
	FileExtension string
	FileSize      uint32
	ChunkSize     uint32
	ChunkInfo     map[uint32]*ChunkMetadata //chunkNumber-->ChunkMetaData

}
