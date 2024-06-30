package peer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"tarun-kavipurapu/stream-files/pkg"
)

type Store struct {
}

func (s *Store) openFile(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}
func (s *Store) createChunkDirectory(hashString string) (string, error) {
	fileDirectory := fmt.Sprintf("chunks/%s", hashString)
	err := os.MkdirAll(fileDirectory, os.ModePerm)
	if err != nil {
		return "", err
	}
	return fileDirectory, nil
}
func (s *Store) writeChunk(fileDirectory string, chunkIndex uint32, r io.Reader, bytesRead int) error {
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
func (s *Store) divideToChunk(file *os.File, chunkSize int, fileDirectory string, peerId string) (error, map[uint32]*pkg.ChunkMetadata) {
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
		chunkMetaData := pkg.ChunkMetadata{
			ChunkIndex: i,
		}

		chunkMap[i] = &chunkMetaData

		if err := s.writeChunk(fileDirectory, i, bytes.NewBuffer(buffer), bytesRead); err != nil {
			return err, nil
		}
	}
	return nil, chunkMap
}
