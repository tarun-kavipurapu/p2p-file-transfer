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

		if err := s.writeChunk(fileDirectory, i, bytes.NewBuffer(buffer)); err != nil {
			return err, nil
		}
	}
	return nil, chunkMap
}

func (s *Store) fetchFileByChunkId(fileId string, chunkIndex uint32) ([]byte, error) {
	chunkName := fmt.Sprintf("chunk_%d.chunk", chunkIndex)
	fileDirectory := fmt.Sprintf("chunks/%s", fileId)

	chunkFilePath := filepath.Join(fileDirectory, chunkName)

	file, err := s.openFile(chunkFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := pkg.ReadBytes(file)
	if err != nil {
		return nil, err
	}
	return data, err

}
