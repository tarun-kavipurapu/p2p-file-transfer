package pkg

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

func HashFile(file *os.File) (string, error) {
	hash := sha256.New()
	defer file.Seek(0, io.SeekStart)

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return "", err
	}
	if _, err := hash.Write(buf.Bytes()); err != nil {
		return "", fmt.Errorf("error updating hash: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// while hashing the chunk we are only hashing the buffer that is read from the File
func HashChunk(r io.Reader) (string, error) {
	hash := sha256.New()

	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func CheckFileHash(file *os.File, expectedHash string) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start of file %s: %w", file.Name(), err)
	}

	hashFromFile, err := HashFile(file)
	if err != nil {
		return fmt.Errorf("failed to hash file %s: %w", file.Name(), err)
	}
	if hashFromFile != expectedHash {
		return errors.New("file is corrupted, hash mismatch")
	}
	return nil
}
