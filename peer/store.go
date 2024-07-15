package peer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"tarun-kavipurapu/p2p-transfer/pkg"
)

type Store struct {
}

func (s *Store) createChunkDirectory(folderName string, hashString string) (string, error) {
	fileDirectory := fmt.Sprintf("%s/%s", folderName, hashString)
	err := os.MkdirAll(fileDirectory, os.ModePerm)
	if err != nil {
		return "", err
	}
	return fileDirectory, nil
}

func (s *Store) writeChunk(fileDirectory string, chunkIndex uint32, r io.Reader) error {
	chunkName := fmt.Sprintf("chunk_%d.chunk", chunkIndex)
	chunkFilePath := filepath.Join(fileDirectory, chunkName)

	writeFile, err := os.Create(chunkFilePath)
	if err != nil {
		return err
	}
	defer writeFile.Close()

	_, err = io.Copy(writeFile, r)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) divideToChunk(file *os.File, chunkSize int, fileDirectory string, peerServerAddr string) (error, map[uint32]*pkg.ChunkMetadata) {
	buffer := make([]byte, chunkSize)
	chunkMap := make(map[uint32]*pkg.ChunkMetadata) // Initialize the map
	for i := uint32(1); ; i++ {
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return err, nil
		}

		if bytesRead == 0 {
			break
		}
		chunkHash, err := pkg.HashChunk(bytes.NewReader(buffer[:bytesRead]))
		if err != nil {
			return err, nil
		}
		chunkMetaData := pkg.ChunkMetadata{
			ChunkHash:      chunkHash,
			ChunkIndex:     i,
			PeersWithChunk: []string{peerServerAddr},
		}
		chunkMap[i] = &chunkMetaData

		if err := s.writeChunk(fileDirectory, i, bytes.NewBuffer(buffer[:bytesRead])); err != nil {
			return err, nil
		}
	}
	return nil, chunkMap
}
