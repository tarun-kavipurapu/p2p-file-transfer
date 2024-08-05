package peer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"tarun-kavipurapu/p2p-transfer/pkg"
)

type Store struct {
}

func (s *Store) ReassembleFile(fileId string, fileName string, fileExtension string, peerAddr string) error {
	dir := fmt.Sprintf("chunks-%s", peerAddr)
	chunkDir := filepath.Join(dir, fileId)
	outputPath := filepath.Join(chunkDir, fmt.Sprintf("%s.%s", fileName, fileExtension))

	// Ensure the downloads directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	// Open the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Get a list of all chunk files
	chunkFiles, err := filepath.Glob(filepath.Join(chunkDir, "chunk_*.chunk"))
	if err != nil {
		return fmt.Errorf("failed to list chunk files: %w", err)
	}

	// Sort the chunk files to ensure correct order
	sort.Slice(chunkFiles, func(i, j int) bool {
		iNum := getChunkNumber(chunkFiles[i])
		jNum := getChunkNumber(chunkFiles[j])
		return iNum < jNum
	})

	// Reassemble the file
	for _, chunkFile := range chunkFiles {
		chunk, err := os.Open(chunkFile)
		if err != nil {
			return fmt.Errorf("failed to open chunk file %s: %w", chunkFile, err)
		}
		defer chunk.Close()

		_, err = io.Copy(outputFile, chunk)
		if err != nil {
			return fmt.Errorf("failed to write chunk %s to output file: %w", chunkFile, err)
		}
	}

	return nil
}
func getChunkNumber(filename string) int {
	base := filepath.Base(filename)
	numStr := strings.TrimPrefix(strings.TrimSuffix(base, ".chunk"), "chunk_")
	num := 0
	fmt.Sscanf(numStr, "%d", &num)
	return num
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
